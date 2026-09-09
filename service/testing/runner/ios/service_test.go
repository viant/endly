package ios

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

const devicesJSON = `{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-18-0": [
      {"udid":"BASE-UDID","name":"Endly Base","state":"Shutdown","isAvailable":true}
    ]
  }
}`

const runtimesJSON = `{
  "runtimes": [
    {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-18-0","isAvailable":true}
  ]
}`

type fakeRunner struct {
	mu       sync.Mutex
	commands []mobile.Command
	runHook  func(mobile.Command) (mobile.Result, error)
}

func (f *fakeRunner) Run(_ context.Context, command mobile.Command) (mobile.Result, error) {
	f.mu.Lock()
	f.commands = append(f.commands, command)
	f.mu.Unlock()
	if f.runHook != nil {
		if result, err := f.runHook(command); result.Stdout != "" || result.Stderr != "" || result.DurationMs != 0 || err != nil {
			return result, err
		}
	}
	joined := strings.Join(command.Args, " ")
	switch {
	case joined == "-version":
		return mobile.Result{Stdout: "Xcode 27.0\nBuild version 27A"}, nil
	case joined == "--version":
		return mobile.Result{Stdout: "xcrun version 72."}, nil
	case joined == "simctl list runtimes available --json":
		return mobile.Result{Stdout: runtimesJSON}, nil
	case joined == "simctl list devices available --json":
		return mobile.Result{Stdout: devicesJSON}, nil
	case strings.HasPrefix(joined, "simctl clone "):
		return mobile.Result{Stdout: "CLONE-UDID\n"}, nil
	default:
		return mobile.Result{}, nil
	}
}

func (f *fakeRunner) Start(_ context.Context, _ mobile.Command, _, _ io.Writer) (*mobile.Process, error) {
	done := make(chan error)
	close(done)
	return &mobile.Process{PID: 1, Done: done}, nil
}

func recordedCommands(runner *fakeRunner) string {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	commands := make([]string, 0, len(runner.commands))
	for _, command := range runner.commands {
		commands = append(commands, command.Name+" "+strings.Join(command.Args, " "))
	}
	return strings.Join(commands, "\n")
}

func TestParseSimulators(t *testing.T) {
	simulators, err := parseSimulators(devicesJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(simulators) != 1 || simulators[0].UDID != "BASE-UDID" || simulators[0].Runtime == "" {
		t.Fatalf("unexpected simulators: %+v", simulators)
	}
}

func TestRoutes(t *testing.T) {
	service := newService(&fakeRunner{})
	for _, action := range []string{"doctor", "simulator-start", "simulator-stop", "server-start", "server-stop", "build", "install", "uninstall", "launch", "terminate", "test", "capture-start", "capture-stop", "open", "attach", "run", "repl", "artifact", "close", "cleanup"} {
		if _, err := service.Route(action); err != nil {
			t.Fatalf("route %q was not registered: %v", action, err)
		}
	}
}

func TestOpenRequestAcceptsManagedServer(t *testing.T) {
	request := &OpenRequest{
		Destination: DestinationLease{ID: "lease", Fence: 1, UDID: "SIM-UDID"},
		Server:      ServerHandle{ID: "server", Endpoint: "http://127.0.0.1:4723", Ownership: "managed"},
		BundleID:    "com.example.app",
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestKeepSessionRequiresExternalDurableResources(t *testing.T) {
	request := &OpenRequest{
		Destination: DestinationLease{ID: "lease", Fence: 1, UDID: "SIM-1", PreserveOnRelease: true},
		Server:      ServerHandle{Endpoint: "http://127.0.0.1:4723", Ownership: "external"},
		BundleID:    "com.example.app", DescriptorPath: "/tmp/ios-session.json", KeepSession: true,
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	request.Destination.PreserveOnRelease = false
	if err := request.Validate(); err == nil {
		t.Fatal("expected non-persistent Simulator lease to be rejected for session handoff")
	}
	if err := (&SimulatorStartRequest{BaseName: "Base", CloneName: "Clone", KeepBooted: true}).Validate(); err == nil {
		t.Fatal("expected owned clone to be rejected for KeepBooted")
	}
}

func TestStopOwnedWDARequiresExactHomeAndUDID(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		joined := strings.Join(command.Args, " ")
		if strings.HasSuffix(command.Name, "/ps") || command.Name == "ps" {
			return mobile.Result{Stdout: strings.Join([]string{
				"123 /usr/bin/xcodebuild -project /owned/appium-home/node_modules/appium-webdriveragent/WebDriverAgent.xcodeproj -destination id=OWNED-UDID APPIUM_XCODEBUILD_WDA_MARKER=1",
				"456 /usr/bin/xcodebuild -project /other/appium-home/node_modules/appium-webdriveragent/WebDriverAgent.xcodeproj -destination id=OTHER-UDID APPIUM_XCODEBUILD_WDA_MARKER=1",
			}, "\n")}, nil
		}
		if joined == "-0 123" {
			return mobile.Result{}, errors.New("process exited")
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	err := service.stopOwnedWDA(context.Background(), &iosServer{
		appium:      &mobile.AppiumServer{Ownership: "managed"},
		destination: DestinationLease{UDID: "OWNED-UDID"},
		appiumHome:  "/owned/appium-home",
	})
	if err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	for _, command := range runner.commands {
		commands = append(commands, strings.Join(command.Args, " "))
	}
	all := strings.Join(commands, "\n")
	if !strings.Contains(all, "-TERM 123") || strings.Contains(all, "-TERM 456") {
		t.Fatalf("cleanup targeted the wrong processes:\n%s", all)
	}
}

func TestLaunchAndTerminateIOSApp(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if strings.Contains(strings.Join(command.Args, " "), "simctl launch") {
			return mobile.Result{Stdout: "com.example.shop: 1234\n"}, nil
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	lease := DestinationLease{ID: "lease-launch", Fence: 1, UDID: "SIM-UDID"}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	launched, err := service.launch(ctx, &LaunchRequest{Destination: lease, BundleID: "com.example.shop", Environment: map[string]string{"MODE": "test"}})
	if err != nil {
		t.Fatal(err)
	}
	if launched.PID != 1234 {
		t.Fatalf("unexpected launch: %+v", launched)
	}
	if _, err := service.terminate(ctx, &TerminateRequest{Destination: lease, BundleID: "com.example.shop"}); err != nil {
		t.Fatal(err)
	}
}

func TestUnifiedLogCaptureLifecycle(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	lease := DestinationLease{ID: "lease-capture", Fence: 1, UDID: "SIM-UDID"}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	started, err := service.captureStart(ctx, &CaptureStartRequest{
		Destination: lease,
		Predicate:   `process == "Shop"`,
		LogPath:     filepath.Join(t.TempDir(), "unified-log.ndjson"),
	})
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := service.captureStop(context.Background(), &CaptureStopRequest{Capture: started.Capture})
	if err != nil {
		t.Fatal(err)
	}
	if !stopped.Stopped || stopped.Artifact == nil || stopped.Artifact.Kind != "unifiedLog" {
		t.Fatalf("unexpected capture response: %+v", stopped)
	}
	if len(runner.commands) != 0 {
		// Capture uses Start, not Run; fakeRunner only records Run commands.
		t.Fatalf("unexpected synchronous commands: %+v", runner.commands)
	}
}

func TestXCTestNormalizesXCResultFailure(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		joined := strings.Join(command.Args, " ")
		if strings.HasSuffix(command.Name, "xcodebuild") && strings.Contains(joined, "test-without-building") {
			return mobile.Result{Stdout: "** TEST FAILED **"}, errors.New("xcodebuild exited 65")
		}
		if strings.Contains(joined, "xcresulttool get test-results summary") {
			return mobile.Result{Stdout: `{
              "title":"ShopUITests",
              "result":"Failed",
              "totalTestCount":2,
              "passedTests":1,
              "failedTests":1,
              "skippedTests":0,
              "expectedFailures":0,
              "testFailures":[{
                "testName":"testCheckout()",
                "targetName":"ShopUITests",
                "failureText":"confirmation missing",
                "testIdentifierString":"ShopUITests/testCheckout()"
              }]
            }`}, nil
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	lease := DestinationLease{ID: "lease-xctest", Fence: 1, UDID: "SIM-UDID"}
	service.storeLease(lease)
	response, err := service.test(endly.New().NewContext(nil), &TestRequest{
		Destination: lease, WorkspacePath: "/tmp/Shop.xcworkspace", Scheme: "Shop",
		Mode: "withoutBuilding", ResultBundlePath: "/tmp/Shop.xcresult", DerivedDataPath: "/tmp/DerivedData",
		CollectDiagnostics: "never", TimeoutMs: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.FailedTests != 1 || len(response.Validations) != 1 || !response.Validations[0].HasFailure() {
		t.Fatalf("unexpected XCTest response: %+v", response)
	}
	joined := []string{}
	for _, command := range runner.commands {
		joined = append(joined, command.Name+" "+strings.Join(command.Args, " "))
	}
	all := strings.Join(joined, "\n")
	for _, expected := range []string{
		"-collect-test-diagnostics never", "-derivedDataPath /tmp/DerivedData", "test-without-building",
		"xcresulttool get test-results summary --path /tmp/Shop.xcresult --compact",
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing %q in:\n%s", expected, all)
		}
	}
}

func TestBuildUsesXcodebuildAndDiscoversSimulatorProducts(t *testing.T) {
	derivedData := t.TempDir()
	appPath := filepath.Join(derivedData, "Build", "Products", "Debug-iphonesimulator", "Shop.app")
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if !strings.HasSuffix(command.Name, "xcodebuild") {
			return mobile.Result{}, nil
		}
		if err := os.MkdirAll(appPath, 0o755); err != nil {
			return mobile.Result{}, err
		}
		if err := os.WriteFile(filepath.Join(appPath, "Shop"), []byte("binary"), 0o700); err != nil {
			return mobile.Result{}, err
		}
		return mobile.Result{Stdout: "** BUILD SUCCEEDED **", DurationMs: 42}, nil
	}}
	service := newService(runner)
	lease := DestinationLease{ID: "build-lease", Fence: 1, UDID: "SIM-UDID"}
	service.storeLease(lease)
	request := &BuildRequest{
		WorkspacePath: "/tmp/Shop.xcworkspace", Scheme: "Shop", Configuration: "Debug",
		Destination: lease, DerivedDataPath: derivedData, Mode: "build", TimeoutMs: 1000,
	}
	response, err := service.build(endly.New().NewContext(nil), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 1 || response.Artifacts[0].Kind != "simulatorApp" || response.Artifacts[0].SHA256 == "" {
		t.Fatalf("unexpected products: %+v", response.Artifacts)
	}
	if len(runner.commands) != 1 || !strings.Contains(strings.Join(runner.commands[0].Args, " "), "-destination platform=iOS Simulator,id=SIM-UDID") {
		t.Fatalf("unexpected xcodebuild command: %+v", runner.commands)
	}
}

func TestArchiveAndExportUseSigningSettingsAndDiscoverIPA(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "Fixture.xcarchive")
	exportPath := filepath.Join(directory, "export")
	exportOptions := filepath.Join(directory, "ExportOptions.plist")
	if err := os.WriteFile(exportOptions, []byte("<plist/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		joined := strings.Join(command.Args, " ")
		if !strings.HasSuffix(command.Name, "xcodebuild") {
			return mobile.Result{}, nil
		}
		if strings.Contains(joined, " archive") {
			if err := os.MkdirAll(filepath.Join(archivePath, "Products", "Applications", "Fixture.app"), 0o755); err != nil {
				return mobile.Result{}, err
			}
			if err := os.WriteFile(filepath.Join(archivePath, "Info.plist"), []byte("archive"), 0o600); err != nil {
				return mobile.Result{}, err
			}
		}
		if strings.Contains(joined, "-exportArchive") {
			if err := os.MkdirAll(exportPath, 0o755); err != nil {
				return mobile.Result{}, err
			}
			if err := os.WriteFile(filepath.Join(exportPath, "Fixture.ipa"), []byte("ipa"), 0o600); err != nil {
				return mobile.Result{}, err
			}
		}
		return mobile.Result{Stdout: "ok", DurationMs: 2}, nil
	}}
	service := newService(runner)
	request := &BuildRequest{
		ProjectPath: "/tmp/Fixture.xcodeproj", Scheme: "Fixture", Configuration: "Release",
		Mode: "archiveAndExport", ArchivePath: archivePath, ExportPath: exportPath,
		ExportOptionsPlist: exportOptions, TimeoutMs: 1000,
		Signing: &IOSSigningProfile{
			Style: "manual", TeamID: "TEAM123", Identity: "Apple Distribution",
			ProvisioningProfile: "Fixture Distribution", KeychainPath: "/secure/ci.keychain-db",
		},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	response, err := service.build(endly.New().NewContext(nil), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 2 || response.Artifacts[0].Kind != "xcarchive" || response.Artifacts[1].Kind != "ipa" {
		t.Fatalf("unexpected archive artifacts: %+v", response.Artifacts)
	}
	all := recordedCommands(runner)
	for _, expected := range []string{
		"-destination generic/platform=iOS", "CODE_SIGN_STYLE=Manual", "DEVELOPMENT_TEAM=TEAM123",
		"CODE_SIGN_IDENTITY=Apple Distribution", "PROVISIONING_PROFILE_SPECIFIER=Fixture Distribution",
		"OTHER_CODE_SIGN_FLAGS=--keychain /secure/ci.keychain-db", "-exportArchive", "-exportOptionsPlist " + exportOptions,
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing %q in:\n%s", expected, all)
		}
	}
}

func TestInstallAndUninstallUseExactDestinationAndBundle(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DestinationLease{ID: "lease-install", Fence: 1, UDID: "SIM-UDID", Name: "iPhone"}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	installed, err := service.install(ctx, &InstallRequest{
		Destination: lease,
		App:         Artifact{Kind: "simulatorApp", HostPath: "/tmp/Shop.app"},
		BundleID:    "com.example.shop",
		State:       "preserve",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Installed {
		t.Fatalf("unexpected install response: %+v", installed)
	}
	if _, err := service.uninstall(ctx, &UninstallRequest{Destination: lease, BundleID: "com.example.shop"}); err != nil {
		t.Fatal(err)
	}
	joined := []string{}
	for _, command := range runner.commands {
		joined = append(joined, strings.Join(command.Args, " "))
	}
	all := strings.Join(joined, "\n")
	for _, expected := range []string{
		"simctl install SIM-UDID /tmp/Shop.app",
		"simctl uninstall SIM-UDID com.example.shop",
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing command %q in:\n%s", expected, all)
		}
	}
}

func TestAppiumRunnerFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value interface{}
		switch r.Method + " " + r.URL.Path {
		case "GET /status":
			value = map[string]interface{}{"ready": true}
		case "POST /session":
			value = map[string]interface{}{"sessionId": "backend-1", "capabilities": map[string]interface{}{}}
		case "POST /session/backend-1/elements":
			value = []map[string]interface{}{{"element-6066-11e4-a52e-4f735466cecf": "element-1"}}
		case "GET /session/backend-1/element/element-1/text":
			value = "Welcome"
		case "GET /session/backend-1/screenshot":
			value = base64.StdEncoding.EncodeToString([]byte("png"))
		case "GET /session/backend-1/source":
			value = `<AppiumAUT><XCUIElementTypeStaticText name="greeting" label="Welcome"/></AppiumAUT>`
		case "GET /session/backend-1/contexts":
			value = []string{"NATIVE_APP", "WEBVIEW_com.example.app"}
		case "GET /session/backend-1/context":
			value = "NATIVE_APP"
		case "GET /session/backend-1/orientation":
			value = "PORTRAIT"
		default:
			value = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()

	service := newService(&fakeRunner{})
	lease := DestinationLease{ID: "lease-1", Fence: 1, UDID: "SIM-UDID", Name: "iPhone"}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	serverResponse, err := service.serverStart(ctx, &ServerStartRequest{Destination: lease, Mode: "external", ServerURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	descriptorPath := filepath.Join(t.TempDir(), "opened-session.json")
	opened, err := service.open(ctx, &OpenRequest{
		SessionID:      "ios-test",
		Destination:    lease,
		Server:         serverResponse.Server,
		BundleID:       "com.example.app",
		DescriptorPath: descriptorPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Session.BackendSessionID != "backend-1" {
		t.Fatalf("unexpected session: %+v", opened.Session)
	}
	if descriptor, err := mobile.ReadSessionDescriptor(descriptorPath, "ios"); err != nil || descriptor.BackendSessionID != "backend-1" {
		t.Fatalf("open did not publish a usable descriptor: %+v err=%v", descriptor, err)
	}
	result, err := service.run(ctx, &RunRequest{
		SessionID: opened.Session.ID,
		Commands: []interface{}{
			`greeting = app.getByTestId("greeting").text()`,
			map[string]interface{}{
				"key": "typedGreeting", "locator": map[string]interface{}{"strategy": "accessibilityId", "value": "greeting"},
				"action": map[string]interface{}{"name": "text"},
			},
			`expect(app.getByTestId("greeting")).toHaveText("Welcome", 20)`,
		},
		ActionTimeoutMs: 50,
		PollIntervalMs:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["greeting"] != "Welcome" || result.Data["typedGreeting"] != "Welcome" || len(result.Validations) != 1 || result.Validations[0].PassedCount != 1 {
		t.Fatalf("unexpected run response: %+v", result)
	}
	deviceResult, err := service.run(ctx, &RunRequest{
		SessionID: opened.Session.ID,
		Commands: []interface{}{
			`contexts = device.contexts()`,
			`expect(device.context()).toHaveContext("NATIVE_APP", 20)`,
			`expect(device.orientation()).toHaveOrientation("PORTRAIT", 20)`,
		},
		ActionTimeoutMs: 50, PollIntervalMs: 1,
	})
	if err != nil || len(deviceResult.Validations) != 2 || len(deviceResult.Data["contexts"].([]string)) != 2 {
		t.Fatalf("unexpected device result: %+v, err=%v", deviceResult, err)
	}
	captureLog := filepath.Join(t.TempDir(), "active-unified-log.ndjson")
	if err := os.WriteFile(captureLog, []byte("failure log tail"), 0o600); err != nil {
		t.Fatal(err)
	}
	failureVideo := newFailureVideoCapture(t)
	service.captures["failure-capture"] = &iosCapture{
		handle: CaptureHandle{ID: "failure-capture", Destination: lease},
		log:    &mobile.LoggedProcess{Path: captureLog}, video: failureVideo,
	}
	failureResult, err := service.run(ctx, &RunRequest{
		SessionID:       opened.Session.ID,
		Commands:        []interface{}{`expect(app.getByTestId("greeting")).toHaveText("Missing", 5)`},
		ActionTimeoutMs: 10, PollIntervalMs: 1,
		FailureArtifacts: &mobile.FailureArtifactOptions{Directory: t.TempDir()},
	})
	if err != nil || len(failureResult.Failures) != 1 || len(failureResult.Failures[0].Artifacts) != 5 {
		t.Fatalf("automatic failure evidence missing: %+v, err=%v", failureResult, err)
	}
	evidence, err := service.artifact(ctx, &ArtifactRequest{SessionID: opened.Session.ID, Directory: t.TempDir(), Screenshot: true, PageSource: true, MaxSourceBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Artifacts) != 2 {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	for _, item := range evidence.Artifacts {
		if _, err := os.Stat(item.URL); err != nil {
			t.Fatalf("evidence %s was not written: %v", item.URL, err)
		}
	}
	service.input = strings.NewReader(":status\n:tree Welcome\nreplValue = app.getByTestId(\"greeting\").text()\n:screenshot\n:quit\n")
	replOutput := &bytes.Buffer{}
	service.output = replOutput
	repl, err := service.repl(ctx, &REPLRequest{
		SessionID: opened.Session.ID, ArtifactDirectory: t.TempDir(),
		ActionTimeoutMs: 50, PollIntervalMs: 1, MaxSourceBytes: 1000, MaxTreeNodes: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repl.Result.Commands != 1 || len(repl.Result.Artifacts) != 1 || repl.Result.Data["replValue"] != "Welcome" {
		t.Fatalf("unexpected REPL response: %+v", repl)
	}
	for _, expected := range []string{"ios[ios-test]>", `label="Welcome"`, "replValue = Welcome"} {
		if !strings.Contains(replOutput.String(), expected) {
			t.Fatalf("missing %q in REPL output:\n%s", expected, replOutput.String())
		}
	}
	closed, err := service.close(context.Background(), &CloseRequest{SessionID: opened.Session.ID})
	if err != nil || !closed.Closed {
		t.Fatalf("unexpected close response: %+v, err=%v", closed, err)
	}
	if _, err := os.Stat(descriptorPath); !os.IsNotExist(err) {
		t.Fatalf("closed session descriptor remains: %v", err)
	}
	if _, err := service.serverStop(context.Background(), &ServerStopRequest{Server: serverResponse.Server}); err != nil {
		t.Fatal(err)
	}
}

func newFailureVideoCapture(t *testing.T) *mobile.SegmentedCapture {
	t.Helper()
	directory := t.TempDir()
	capture, err := mobile.StartSegmentedCapture(time.Hour,
		func(_ context.Context, index int) (*mobile.Process, error) {
			done := make(chan error)
			return &mobile.Process{PID: index + 1, Done: done, Stop: func(context.Context) error {
				close(done)
				return nil
			}}, nil
		},
		func(_ context.Context, index int) (string, error) {
			path := filepath.Join(directory, fmt.Sprintf("segment-%d.mp4", index))
			if err := os.WriteFile(path, []byte("video"), 0o600); err != nil {
				return "", err
			}
			return path, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { capture.Stop() })
	return capture
}

func TestAttachReconnectsAcrossServiceInstances(t *testing.T) {
	deleteCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value interface{}
		switch r.Method + " " + r.URL.Path {
		case "GET /session/backend-remote":
			value = map[string]interface{}{"capabilities": map[string]interface{}{"platformName": "iOS", "appium:udid": "SIM-REMOTE"}}
		case "POST /session/backend-remote/elements":
			value = []map[string]interface{}{{mobile.W3CElementKey: "element-1"}}
		case "GET /session/backend-remote/element/element-1/text":
			value = "Remote"
		case "DELETE /session/backend-remote":
			deleteCount++
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	descriptorPath := filepath.Join(t.TempDir(), "ios-session.json")
	if err := mobile.WriteSessionDescriptor(descriptorPath, &mobile.SessionDescriptor{
		Platform: "ios", SessionID: "remote", BackendSessionID: "backend-remote",
		Endpoint: server.URL, TargetID: "SIM-REMOTE",
	}); err != nil {
		t.Fatal(err)
	}

	firstProcess := newService(&fakeRunner{})
	firstContext := endly.New().NewContext(nil)
	attached, err := firstProcess.attach(firstContext, &AttachRequest{DescriptorPath: descriptorPath})
	if err != nil {
		t.Fatal(err)
	}
	result, err := firstProcess.run(firstContext, &RunRequest{
		SessionID: attached.Session.ID, Commands: []interface{}{`remote = app.getByTestId("title").text()`},
		ActionTimeoutMs: 20, PollIntervalMs: 1,
	})
	if err != nil || result.Data["remote"] != "Remote" {
		t.Fatalf("attached run failed: result=%+v err=%v", result, err)
	}
	detached, err := firstProcess.close(context.Background(), &CloseRequest{SessionID: attached.Session.ID})
	if err != nil || !strings.Contains(detached.Warning, "backend session remains open") || deleteCount != 0 {
		t.Fatalf("safe detach failed: response=%+v deletes=%d err=%v", detached, deleteCount, err)
	}

	secondProcess := newService(&fakeRunner{})
	secondContext := endly.New().NewContext(nil)
	secondProcess.input = strings.NewReader(":status\n:quit\n")
	secondOutput := &bytes.Buffer{}
	secondProcess.output = secondOutput
	reconnected, err := secondProcess.repl(secondContext, &REPLRequest{
		Attach: &AttachRequest{DescriptorPath: descriptorPath, SessionID: "remote-owner", TakeOwnership: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(secondOutput.String(), `"attached": true`) {
		t.Fatalf("inline attached REPL did not report attachment:\n%s", secondOutput.String())
	}
	closed, err := secondProcess.close(context.Background(), &CloseRequest{SessionID: reconnected.SessionID})
	if err != nil || !closed.Closed || deleteCount != 1 {
		t.Fatalf("owned reconnect close failed: response=%+v deletes=%d err=%v", closed, deleteCount, err)
	}
	if _, err := os.Stat(descriptorPath); !os.IsNotExist(err) {
		t.Fatalf("owned close left descriptor behind: %v", err)
	}
}

func TestDoctorAndOwnedCloneLifecycle(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	ctx := endly.New().NewContext(nil)
	doctor, err := service.doctor(ctx, &DoctorRequest{Required: []string{"xcodebuild", "simulator-runtime"}})
	if err != nil {
		t.Fatal(err)
	}
	if !doctor.Ready || len(doctor.Runtimes) != 1 || len(doctor.Simulators) != 1 {
		t.Fatalf("unexpected doctor response: %+v", doctor)
	}

	request := &SimulatorStartRequest{BaseName: "Endly Base", CloneName: "Endly Clone", Erase: true, BootTimeoutMs: 1000}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	started, err := service.simulatorStart(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !started.Lease.OwnedClone || started.Lease.UDID != "CLONE-UDID" || started.Lease.ProcessLease == nil {
		t.Fatalf("unexpected lease: %+v", started.Lease)
	}
	stopped, err := service.simulatorStop(ctx, &SimulatorStopRequest{Lease: started.Lease})
	if err != nil {
		t.Fatal(err)
	}
	if !stopped.Shutdown || !stopped.Deleted {
		t.Fatalf("expected owned clone shutdown+delete: %+v", stopped)
	}
	if _, err := os.Stat(started.Lease.ProcessLease.Path); !os.IsNotExist(err) {
		t.Fatalf("persistent destination lease remains after stop: %v", err)
	}
}

func TestExistingSimulatorCanRemainBootedForSessionHandoff(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	ctx := endly.New().NewContext(nil)
	request := &SimulatorStartRequest{BaseName: "Endly Base", KeepBooted: true, BootTimeoutMs: 1000}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	started, err := service.simulatorStart(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !started.Lease.PreserveOnRelease || started.Lease.OwnedClone {
		t.Fatalf("unexpected handoff lease: %+v", started.Lease)
	}
	stopped, err := service.simulatorStop(ctx, &SimulatorStopRequest{Lease: started.Lease})
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Shutdown || !strings.Contains(stopped.Warning, "left booted") {
		t.Fatalf("Simulator was not preserved: %+v", stopped)
	}
	if strings.Contains(recordedCommands(runner), "simctl shutdown BASE-UDID") {
		t.Fatalf("preserved Simulator was shut down:\n%s", recordedCommands(runner))
	}
}

func TestSimulatorStopRejectsFenceMismatch(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DestinationLease{ID: "lease-1", Fence: 2, UDID: "CLONE-UDID", OwnedClone: true}
	service.storeLease(lease)
	requestLease := lease
	requestLease.Fence = 1
	_, err := service.simulatorStop(endly.New().NewContext(nil), &SimulatorStopRequest{Lease: requestLease})
	if err == nil || !strings.Contains(err.Error(), "fence mismatch") {
		t.Fatalf("expected fence mismatch, got %v", err)
	}
}
