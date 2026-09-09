package android

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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

type fakeRunner struct {
	mu       sync.Mutex
	commands []mobile.Command
	started  []mobile.Command
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
	case joined == "version":
		return mobile.Result{Stdout: "Android Debug Bridge version 1.0.41\n"}, nil
	case joined == "devices -l":
		return mobile.Result{Stdout: "List of devices attached\nemulator-5556 device product:sdk model:Pixel transport_id:1\n"}, nil
	case joined == "-version":
		return mobile.Result{Stdout: "Android emulator version 36.1\n"}, nil
	case joined == "-list-avds":
		return mobile.Result{Stdout: "pixel_api_35\n"}, nil
	case strings.Contains(joined, "getprop sys.boot_completed"):
		return mobile.Result{Stdout: "1\n"}, nil
	default:
		return mobile.Result{}, nil
	}
}

func (f *fakeRunner) Start(_ context.Context, command mobile.Command, _, _ io.Writer) (*mobile.Process, error) {
	f.mu.Lock()
	f.started = append(f.started, command)
	f.mu.Unlock()
	done := make(chan error)
	close(done)
	return &mobile.Process{PID: 4242, Done: done}, nil
}

func TestParseADBDevices(t *testing.T) {
	devices := parseADBDevices("List of devices attached\nABC unauthorized usb:1-2\nemulator-5554 device product:sdk model:Pixel_8\n")
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[1].Serial != "emulator-5554" || devices[1].State != "device" {
		t.Fatalf("unexpected parsed device: %+v", devices[1])
	}
}

func TestRoutes(t *testing.T) {
	service := newService(&fakeRunner{})
	for _, action := range []string{"doctor", "device-start", "device-stop", "server-start", "server-stop", "build", "install", "uninstall", "launch", "terminate", "test", "capture-start", "capture-stop", "open", "attach", "run", "repl", "artifact", "close", "cleanup"} {
		if _, err := service.Route(action); err != nil {
			t.Fatalf("route %q was not registered: %v", action, err)
		}
	}
}

func TestOpenRequestAcceptsManagedServer(t *testing.T) {
	request := &OpenRequest{
		Lease:   DeviceLease{ID: "lease", Fence: 1, Serial: "emulator-5554"},
		Server:  ServerHandle{ID: "server", Endpoint: "http://127.0.0.1:4723", Ownership: "managed"},
		Package: "com.example.app", TestIDStrategy: "resourceId",
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestKeepSessionRequiresExternalDurableResources(t *testing.T) {
	request := &OpenRequest{
		Lease:   DeviceLease{ID: "lease", Fence: 1, Serial: "device-1"},
		Server:  ServerHandle{Endpoint: "http://127.0.0.1:4723", Ownership: "external"},
		Package: "com.example.app", TestIDStrategy: "accessibilityId",
		DescriptorPath: "/tmp/android-session.json", KeepSession: true,
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	request.Lease.Owned = true
	if err := request.Validate(); err == nil {
		t.Fatal("expected owned emulator to be rejected for session handoff")
	}
}

func TestLaunchAndTerminateAndroidApp(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if strings.Contains(strings.Join(command.Args, " "), "resolve-activity") {
			return mobile.Result{Stdout: "com.example.app/.MainActivity\n"}, nil
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-launch", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	launched, err := service.launch(ctx, &LaunchRequest{Lease: lease, Package: "com.example.app"})
	if err != nil {
		t.Fatal(err)
	}
	if launched.Component != "com.example.app/.MainActivity" {
		t.Fatalf("unexpected launch: %+v", launched)
	}
	if _, err := service.terminate(ctx, &TerminateRequest{Lease: lease, Package: "com.example.app"}); err != nil {
		t.Fatal(err)
	}
}

func TestLogcatCaptureLifecycle(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	lease := DeviceLease{ID: "lease-capture", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	started, err := service.captureStart(ctx, &CaptureStartRequest{Lease: lease, LogPath: filepath.Join(t.TempDir(), "logcat.txt"), Clear: true})
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := service.captureStop(context.Background(), &CaptureStopRequest{Capture: started.Capture})
	if err != nil {
		t.Fatal(err)
	}
	if !stopped.Stopped || stopped.Artifact == nil || stopped.Artifact.Kind != "logcat" {
		t.Fatalf("unexpected capture response: %+v", stopped)
	}
	if len(runner.started) != 1 || !strings.Contains(strings.Join(runner.started[0].Args, " "), "logcat -v threadtime") {
		t.Fatalf("unexpected capture command: %+v", runner.started)
	}
}

func TestInstrumentationInstallsBothPackagesAndReportsFailure(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if strings.Contains(strings.Join(command.Args, " "), "am instrument") {
			return mobile.Result{Stdout: strings.Join([]string{
				"INSTRUMENTATION_STATUS: class=com.example.CheckoutTest",
				"INSTRUMENTATION_STATUS: test=canCheckout",
				"INSTRUMENTATION_STATUS: stack=expected confirmation",
				"INSTRUMENTATION_STATUS_CODE: -2",
				"INSTRUMENTATION_CODE: -1",
			}, "\n")}, nil
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-test", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	response, err := service.test(endly.New().NewContext(nil), &TestRequest{
		Lease: lease, AppAPKPath: "/tmp/app.apk", TestAPKPath: "/tmp/app-test.apk",
		TestPackage: "com.example.test", Runner: "androidx.test.runner.AndroidJUnitRunner", TimeoutMs: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Failed != 1 || len(response.Validations) != 1 || !response.Validations[0].HasFailure() {
		t.Fatalf("unexpected instrumentation response: %+v", response)
	}
	joined := []string{}
	for _, command := range runner.commands {
		joined = append(joined, strings.Join(command.Args, " "))
	}
	all := strings.Join(joined, "\n")
	for _, expected := range []string{
		"install -r -t /tmp/app.apk",
		"install -r -t /tmp/app-test.apk",
		"am instrument -w -r com.example.test/androidx.test.runner.AndroidJUnitRunner",
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing %q in:\n%s", expected, all)
		}
	}
}

func TestBuildUsesGradleWrapperAndDiscoversArtifacts(t *testing.T) {
	projectDir := t.TempDir()
	wrapper := filepath.Join(projectDir, "gradlew")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(projectDir, "app", "build", "outputs", "apk", "qaDebug", "app-qa-debug.apk")
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if command.Name != wrapper {
			return mobile.Result{}, nil
		}
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			return mobile.Result{}, err
		}
		if err := os.WriteFile(artifactPath, []byte("apk"), 0o600); err != nil {
			return mobile.Result{}, err
		}
		return mobile.Result{Stdout: "BUILD SUCCESSFUL", DurationMs: 12}, nil
	}}
	service := newService(runner)
	request := &BuildRequest{ProjectDir: projectDir, Module: "app", Variant: "qaDebug", Tasks: []string{"assembleQaDebug"}, TimeoutMs: 1000}
	response, err := service.build(endly.New().NewContext(nil), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 1 || response.Artifacts[0].Kind != "appAPK" || response.Artifacts[0].SHA256 == "" {
		t.Fatalf("unexpected artifacts: %+v", response.Artifacts)
	}
	if len(runner.commands) != 1 || !containsArgs(runner.commands[0].Args, "--console=plain", "assembleQaDebug") {
		t.Fatalf("unexpected Gradle command: %+v", runner.commands)
	}
}

func TestInstallAndUninstallUseExactLeaseAndPackage(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-install", Fence: 1, Serial: "emulator-5554", Owned: true, AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	installed, err := service.install(ctx, &InstallRequest{
		Lease: lease, APKPath: "/tmp/app.apk", Package: "com.example.app", State: "cleanData", GrantAll: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Installed {
		t.Fatalf("unexpected install response: %+v", installed)
	}
	if _, err := service.uninstall(ctx, &UninstallRequest{Lease: lease, Package: "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	joined := []string{}
	for _, command := range runner.commands {
		joined = append(joined, strings.Join(command.Args, " "))
	}
	all := strings.Join(joined, "\n")
	for _, expected := range []string{
		"-s emulator-5554 install -r -g /tmp/app.apk",
		"-s emulator-5554 shell pm clear com.example.app",
		"-s emulator-5554 uninstall com.example.app",
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing command %q in:\n%s", expected, all)
		}
	}
}

func TestInstallSplitAPKs(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-split", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	response, err := service.install(endly.New().NewContext(nil), &InstallRequest{
		Lease: lease, APKPaths: []string{"/tmp/base.apk", "/tmp/config.arm64.apk"},
		Package: "com.example.split", State: "preserve",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 2 {
		t.Fatalf("unexpected split response: %+v", response)
	}
	all := recordedCommands(runner)
	if !strings.Contains(all, "install-multiple -r /tmp/base.apk /tmp/config.arm64.apk") {
		t.Fatalf("split APK command missing:\n%s", all)
	}
}

func TestInstallAABUsesPasswordFiles(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-aab", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	directory := t.TempDir()
	bundletool := filepath.Join(directory, "bundletool.jar")
	java := filepath.Join(directory, "java")
	for path, mode := range map[string]os.FileMode{bundletool: 0o600, java: 0o700} {
		if err := os.WriteFile(path, []byte("fixture"), mode); err != nil {
			t.Fatal(err)
		}
	}
	request := &InstallRequest{
		Lease: lease, AABPath: "/tmp/app.aab", BundletoolPath: bundletool, JavaPath: java,
		Package: "com.example.bundle", State: "preserve",
		Signing: &AndroidSigningProfile{
			KeystorePath: "/secure/release.jks", KeyAlias: "release",
			StorePasswordFile: "/secure/store.pass", KeyPasswordFile: "/secure/key.pass",
		},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.install(endly.New().NewContext(nil), request); err != nil {
		t.Fatal(err)
	}
	all := recordedCommands(runner)
	for _, expected := range []string{
		"-jar " + bundletool + " build-apks --bundle=/tmp/app.aab",
		"--ks-pass=file:/secure/store.pass", "--key-pass=file:/secure/key.pass",
		"-jar " + bundletool + " install-apks", "--device-id=emulator-5554",
	} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing %q in:\n%s", expected, all)
		}
	}
}

func recordedCommands(runner *fakeRunner) string {
	lines := []string{}
	for _, command := range runner.commands {
		lines = append(lines, command.Name+" "+strings.Join(command.Args, " "))
	}
	return strings.Join(lines, "\n")
}

func TestFreshInstallSkipsUninstallWhenPackageIsAbsent(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-fresh", Fence: 1, Serial: "emulator-5554", AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	if _, err := service.install(endly.New().NewContext(nil), &InstallRequest{
		Lease: lease, APKPath: "/tmp/app.apk", Package: "com.example.absent", State: "freshInstall",
	}); err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	for _, command := range runner.commands {
		commands = append(commands, strings.Join(command.Args, " "))
	}
	all := strings.Join(commands, "\n")
	if !strings.Contains(all, "shell pm list packages --user 0 com.example.absent") {
		t.Fatalf("package existence was not checked:\n%s", all)
	}
	if strings.Contains(all, "uninstall com.example.absent") {
		t.Fatalf("absent package should not be uninstalled:\n%s", all)
	}
	if !strings.Contains(all, "install /tmp/app.apk") {
		t.Fatalf("APK was not installed:\n%s", all)
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
			value = `<hierarchy><node class="android.widget.TextView" text="Welcome"/></hierarchy>`
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
	lease := DeviceLease{ID: "lease-1", Fence: 1, Serial: "emulator-5554", Owned: true}
	service.storeLease(lease)
	ctx := endly.New().NewContext(nil)
	serverResponse, err := service.serverStart(ctx, &ServerStartRequest{Lease: lease, Mode: "external", ServerURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	descriptorPath := filepath.Join(t.TempDir(), "opened-session.json")
	opened, err := service.open(ctx, &OpenRequest{
		SessionID:      "android-test",
		Lease:          lease,
		Server:         serverResponse.Server,
		Package:        "com.example.app",
		TestIDStrategy: "accessibilityId",
		DescriptorPath: descriptorPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Session.BackendSessionID != "backend-1" {
		t.Fatalf("unexpected session: %+v", opened.Session)
	}
	if descriptor, err := mobile.ReadSessionDescriptor(descriptorPath, "android"); err != nil || descriptor.BackendSessionID != "backend-1" {
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
	captureLog := filepath.Join(t.TempDir(), "active-logcat.txt")
	if err := os.WriteFile(captureLog, []byte("failure log tail"), 0o600); err != nil {
		t.Fatal(err)
	}
	failureVideo := newFailureVideoCapture(t)
	service.captures["failure-capture"] = &androidCapture{
		handle: CaptureHandle{ID: "failure-capture", Lease: lease},
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
	for _, expected := range []string{"android[android-test]>", `text="Welcome"`, "replValue = Welcome"} {
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
			value = map[string]interface{}{"capabilities": map[string]interface{}{"platformName": "Android", "appium:udid": "device-1"}}
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
	descriptorPath := filepath.Join(t.TempDir(), "android-session.json")
	if err := mobile.WriteSessionDescriptor(descriptorPath, &mobile.SessionDescriptor{
		Platform: "android", SessionID: "remote", BackendSessionID: "backend-remote",
		Endpoint: server.URL, TargetID: "device-1", TestIDStrategy: "accessibilityId",
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

func TestDoctorAndOwnedEmulatorLifecycle(t *testing.T) {
	sdkRoot := fakeSDK(t)
	runner := &fakeRunner{}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	ctx := endly.New().NewContext(nil)

	doctorRequest := &DoctorRequest{AndroidSDKRoot: sdkRoot, Required: []string{"adb", "emulator"}}
	if err := doctorRequest.Init(); err != nil {
		t.Fatal(err)
	}
	doctor, err := service.doctor(ctx, doctorRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !doctor.Ready || len(doctor.AVDs) != 1 || len(doctor.Devices) != 1 {
		t.Fatalf("unexpected doctor response: %+v", doctor)
	}

	animations := false
	startRequest := &DeviceStartRequest{
		AndroidSDKRoot: sdkRoot,
		AVD:            "pixel_api_35",
		Port:           5554,
		Animations:     &animations,
		BootTimeoutMs:  1000,
		PollIntervalMs: 1,
	}
	if err := startRequest.Validate(); err != nil {
		t.Fatal(err)
	}
	started, err := service.deviceStart(ctx, startRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !started.Lease.Owned || started.Lease.Serial != "emulator-5554" || started.Lease.PID != 4242 || started.Lease.ProcessLease == nil {
		t.Fatalf("unexpected lease: %+v", started.Lease)
	}
	if len(runner.started) != 1 || !containsArgs(runner.started[0].Args, "-avd", "pixel_api_35", "-port", "5554") {
		t.Fatalf("unexpected emulator start: %+v", runner.started)
	}

	stopped, err := service.deviceStop(ctx, &DeviceStopRequest{Lease: started.Lease})
	if err != nil {
		t.Fatal(err)
	}
	if !stopped.Stopped {
		t.Fatalf("expected owned emulator to stop: %+v", stopped)
	}
	if _, err := os.Stat(started.Lease.ProcessLease.Path); !os.IsNotExist(err) {
		t.Fatalf("persistent device lease remains after stop: %v", err)
	}
}

func TestDeviceStopRejectsFenceMismatch(t *testing.T) {
	runner := &fakeRunner{}
	service := newService(runner)
	lease := DeviceLease{ID: "lease-1", Fence: 2, Serial: "emulator-5554", Owned: true, AndroidSDKRoot: fakeSDK(t)}
	service.storeLease(lease)
	requestLease := lease
	requestLease.Fence = 1
	_, err := service.deviceStop(endly.New().NewContext(nil), &DeviceStopRequest{Lease: requestLease})
	if err == nil || !strings.Contains(err.Error(), "fence mismatch") {
		t.Fatalf("expected fence mismatch, got %v", err)
	}
}

func fakeSDK(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{"platform-tools/adb", "emulator/emulator"} {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func containsArgs(args []string, expected ...string) bool {
	return strings.Contains(strings.Join(args, "\x00"), strings.Join(expected, "\x00"))
}
