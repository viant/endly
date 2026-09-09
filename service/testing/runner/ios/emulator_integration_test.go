package ios

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func TestIOSSimulatorIntegration(t *testing.T) {
	if os.Getenv("ENDLY_IOS_EMULATOR_INTEGRATION") != "1" {
		t.Skip("set ENDLY_IOS_EMULATOR_INTEGRATION=1 to run the real Simulator integration")
	}
	projectPath := os.Getenv("ENDLY_IOS_TEST_PROJECT")
	bundleID := os.Getenv("ENDLY_IOS_TEST_BUNDLE_ID")
	if projectPath == "" {
		projectPath = filepath.Join("test", "fixture", "FixtureApp.xcodeproj")
	}
	if bundleID == "" {
		bundleID = "com.viant.endly.mobilefixture"
	}
	deviceType := os.Getenv("ENDLY_IOS_TEST_DEVICE_TYPE")
	if deviceType == "" {
		deviceType = "com.apple.CoreSimulator.SimDeviceType.iPhone-17-Pro"
	}
	scheme := os.Getenv("ENDLY_IOS_TEST_SCHEME")
	if scheme == "" {
		scheme = "FixtureApp"
	}

	service := newService(mobile.OSRunner{})
	ctx := endly.New().NewContext(nil)
	doctorRequest := &DoctorRequest{Required: []string{"xcodebuild", "simulator-runtime"}}
	if err := doctorRequest.Init(); err != nil {
		t.Fatal(err)
	}
	doctor, err := service.doctor(ctx, doctorRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !doctor.Ready || len(doctor.Runtimes) == 0 {
		t.Fatalf("iOS Simulator host is not ready: %+v", doctor.Checks)
	}
	runtimeID := doctor.Runtimes[len(doctor.Runtimes)-1]

	started, err := service.simulatorStart(ctx, &SimulatorStartRequest{
		DeviceType: deviceType, Runtime: runtimeID, Erase: false, BootTimeoutMs: 180_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if errors := service.cleanupStack(ctx).Close(context.Background()); len(errors) > 0 {
			t.Errorf("cleanup errors: %+v", errors)
		}
	}()

	derivedData := filepath.Join(t.TempDir(), "DerivedData")
	built, err := service.build(ctx, &BuildRequest{
		ProjectPath: projectPath, Scheme: scheme, Configuration: "Debug",
		Destination: started.Lease, DerivedDataPath: derivedData, Mode: "build", TimeoutMs: 20 * 60 * 1000,
	})
	if err != nil {
		if built == nil {
			t.Fatal(err)
		}
		t.Fatalf("%v\nstdout:\n%s\nstderr:\n%s", err, built.Stdout, built.Stderr)
	}
	var app Artifact
	for _, candidate := range built.Artifacts {
		if candidate.Kind == "simulatorApp" && !strings.Contains(candidate.HostPath, "Runner.app") {
			app = candidate
			break
		}
	}
	if app.HostPath == "" {
		t.Fatalf("no unambiguous Simulator app in artifacts: %+v", built.Artifacts)
	}
	if _, err := service.install(ctx, &InstallRequest{Destination: started.Lease, App: app, BundleID: bundleID, State: "freshInstall"}); err != nil {
		t.Fatal(err)
	}
	appiumEnabled := os.Getenv("ENDLY_IOS_APPIUM_INTEGRATION") == "1"
	var appiumSession *OpenResponse
	if appiumEnabled {
		appiumExecutable := os.Getenv("ENDLY_IOS_APPIUM_EXECUTABLE")
		appiumHome := os.Getenv("ENDLY_IOS_APPIUM_HOME")
		if appiumExecutable == "" || appiumHome == "" {
			t.Fatal("ENDLY_IOS_APPIUM_EXECUTABLE and ENDLY_IOS_APPIUM_HOME are required for Appium integration")
		}
		appiumLog := filepath.Join(t.TempDir(), "appium.log")
		server, err := service.serverStart(ctx, &ServerStartRequest{
			Destination: started.Lease, Mode: "managed", Executable: appiumExecutable,
			AppiumHome: appiumHome, Address: "127.0.0.1", Port: 4723,
			LogPath: appiumLog, StartupTimeoutMs: 30_000,
		})
		if err != nil {
			t.Fatalf("start Appium: %v\n%s", err, readDiagnostic(appiumLog))
		}
		appiumSession, err = service.open(ctx, &OpenRequest{
			SessionID: "ios-fixture", Destination: started.Lease, Server: server.Server, BundleID: bundleID,
		})
		if err != nil {
			t.Fatalf("open XCUITest session: %v\n%s", err, readDiagnostic(appiumLog))
		}
		run, err := service.run(ctx, &RunRequest{
			SessionID: appiumSession.Session.ID,
			Commands: []interface{}{
				`expect(app.getByTestId("status")).toHaveText("Endly Mobile Runner", 60000)`,
				`staticTextCount = app.getByType("XCUIElementTypeStaticText").count()`,
				`firstStaticText = app.getByType("XCUIElementTypeStaticText").first().text()`,
				map[string]interface{}{
					"key":     "typedStatus",
					"locator": map[string]interface{}{"strategy": "accessibilityId", "value": "status"},
					"action":  map[string]interface{}{"name": "text"},
				},
				`contexts = device.contexts()`,
				`expect(device.context()).toHaveContext("NATIVE_APP", 10000)`,
				`expect(device.orientation()).toHaveOrientation("PORTRAIT", 10000)`,
				`app.getByTestId("increment").tap()`,
				`expect(app.getByTestId("count")).toHaveText("Count: 1", 10000)`,
				`app.getByTestId("increment").doubleTap()`,
				`expect(app.getByTestId("count")).toHaveText("Count: 2", 10000)`,
			},
			ActionTimeoutMs: 60_000,
			PollIntervalMs:  200,
		})
		if err != nil {
			t.Fatalf("run iOS DSL: %v\n%s", err, readDiagnostic(appiumLog))
		}
		for _, validation := range run.Validations {
			if validation.HasFailure() {
				t.Fatalf("iOS DSL assertion failed: %s\n%s", validation.Report(), readDiagnostic(appiumLog))
			}
		}
		if count, ok := run.Data["staticTextCount"].(int); !ok || count < 2 || run.Data["typedStatus"] != "Endly Mobile Runner" {
			t.Fatalf("unexpected collection/typed data: %+v", run.Data)
		}
		appiumEvidence, err := service.artifact(ctx, &ArtifactRequest{
			SessionID: appiumSession.Session.ID, Directory: t.TempDir(), Screenshot: true, PageSource: true,
		})
		if err != nil || len(appiumEvidence.Artifacts) != 2 {
			t.Fatalf("Appium evidence failed: response=%+v err=%v", appiumEvidence, err)
		}
	}
	videoDirectory := t.TempDir()
	capture, err := service.captureStart(ctx, &CaptureStartRequest{
		Destination:    started.Lease,
		Predicate:      `process == "FixtureApp"`,
		LogPath:        filepath.Join(t.TempDir(), "simulator.ndjson"),
		Video:          true,
		VideoDirectory: videoDirectory,
		SegmentMs:      2_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !appiumEnabled {
		launched, err := service.launch(ctx, &LaunchRequest{Destination: started.Lease, BundleID: bundleID})
		if err != nil {
			t.Fatal(err)
		}
		if launched.PID <= 0 {
			t.Fatalf("Simulator launch did not return a PID: %+v", launched)
		}
	}
	time.Sleep(3 * time.Second)
	evidence, err := service.artifact(ctx, &ArtifactRequest{
		Destination: &started.Lease,
		Directory:   t.TempDir(),
		Screenshot:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Artifacts) != 1 || evidence.Artifacts[0].Size == 0 {
		t.Fatalf("Simulator screenshot was empty: %+v", evidence)
	}
	if _, err := service.terminate(ctx, &TerminateRequest{Destination: started.Lease, BundleID: bundleID}); err != nil {
		t.Fatal(err)
	}
	if stopped, err := service.captureStop(context.Background(), &CaptureStopRequest{Capture: capture.Capture}); err != nil || !stopped.Stopped || !hasVideoEvidence(stopped.Artifacts) {
		t.Fatalf("capture stop failed: response=%+v err=%v", stopped, err)
	}
}

func hasVideoEvidence(artifacts []*mobile.Evidence) bool {
	for _, artifact := range artifacts {
		if artifact != nil && artifact.Kind == "video" && artifact.Size > 0 {
			return true
		}
	}
	return false
}

func readDiagnostic(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	const max = 20_000
	if len(data) > max {
		data = data[len(data)-max:]
	}
	return string(data)
}
