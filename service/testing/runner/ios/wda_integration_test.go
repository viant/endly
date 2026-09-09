//go:build darwin

package ios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func TestIOSWDAModesIntegration(t *testing.T) {
	if os.Getenv("ENDLY_IOS_WDA_MODES_INTEGRATION") != "1" {
		t.Skip("set ENDLY_IOS_WDA_MODES_INTEGRATION=1 to validate managed, external, prebuilt, and preinstalled WDA")
	}
	appiumExecutable := os.Getenv("ENDLY_IOS_APPIUM_EXECUTABLE")
	appiumHome := os.Getenv("ENDLY_IOS_APPIUM_HOME")
	if appiumExecutable == "" || appiumHome == "" {
		t.Fatal("ENDLY_IOS_APPIUM_EXECUTABLE and ENDLY_IOS_APPIUM_HOME are required")
	}
	projectPath := os.Getenv("ENDLY_IOS_TEST_PROJECT")
	if projectPath == "" {
		projectPath = filepath.Join("test", "fixture", "FixtureApp.xcodeproj")
	}
	bundleID := os.Getenv("ENDLY_IOS_TEST_BUNDLE_ID")
	if bundleID == "" {
		bundleID = "com.viant.endly.mobilefixture"
	}
	scheme := os.Getenv("ENDLY_IOS_TEST_SCHEME")
	if scheme == "" {
		scheme = "FixtureApp"
	}
	deviceType := os.Getenv("ENDLY_IOS_TEST_DEVICE_TYPE")
	if deviceType == "" {
		deviceType = "com.apple.CoreSimulator.SimDeviceType.iPhone-17-Pro"
	}

	service := newService(mobile.OSRunner{})
	ctx := endly.New().NewContext(nil)
	cleanup := service.cleanupStack(ctx)
	cleanupComplete := false
	defer func() {
		if !cleanupComplete {
			if cleanupErrors := cleanup.Close(context.Background()); len(cleanupErrors) > 0 {
				t.Errorf("WDA mode fallback cleanup: %+v", cleanupErrors)
			}
		}
	}()
	doctor, err := service.doctor(ctx, &DoctorRequest{Required: []string{"xcodebuild", "simulator-runtime"}})
	if err != nil || !doctor.Ready || len(doctor.Runtimes) == 0 {
		t.Fatalf("iOS host not ready: response=%+v err=%v", doctor, err)
	}
	destination, err := service.simulatorStart(ctx, &SimulatorStartRequest{
		DeviceType: deviceType, Runtime: doctor.Runtimes[len(doctor.Runtimes)-1], BootTimeoutMs: 240_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	derivedData := filepath.Join(t.TempDir(), "AppDerivedData")
	built, err := service.build(ctx, &BuildRequest{
		ProjectPath: projectPath, Scheme: scheme, Configuration: "Debug", Destination: destination.Lease,
		DerivedDataPath: derivedData, Mode: "build", TimeoutMs: 20 * 60 * 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	var app Artifact
	for _, artifact := range built.Artifacts {
		if artifact.Kind == "simulatorApp" && !strings.Contains(artifact.HostPath, "Runner.app") {
			app = artifact
			break
		}
	}
	if app.HostPath == "" {
		t.Fatalf("no Simulator app in build artifacts: %+v", built.Artifacts)
	}
	if _, err := service.install(ctx, &InstallRequest{Destination: destination.Lease, App: app, BundleID: bundleID, State: "freshInstall"}); err != nil {
		t.Fatal(err)
	}
	server, err := service.serverStart(ctx, &ServerStartRequest{
		Destination: destination.Lease, Mode: "managed", Executable: appiumExecutable, AppiumHome: appiumHome,
		Address: "127.0.0.1", Port: 4723, LogPath: filepath.Join(t.TempDir(), "appium.log"), StartupTimeoutMs: 45_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	wdaDerivedData := filepath.Join(t.TempDir(), "WDADerivedData")
	useNew := false
	modes := []struct {
		name    string
		options *WDAOptions
	}{
		{name: "managed", options: &WDAOptions{
			Mode: "managed", DerivedDataPath: wdaDerivedData, LocalPort: 8100, MJPEGServerPort: 9100,
			UseNewWDA: &useNew, LaunchTimeoutMs: 120_000, ConnectionTimeoutMs: 120_000,
		}},
		{name: "external", options: &WDAOptions{
			Mode: "external", WebDriverAgentURL: "http://127.0.0.1:8100",
		}},
		{name: "prebuilt", options: &WDAOptions{
			Mode: "prebuilt", DerivedDataPath: wdaDerivedData, LocalPort: 8100, MJPEGServerPort: 9100,
			UseNewWDA: &useNew, LaunchTimeoutMs: 120_000,
		}},
	}
	preinstalled := &WDAOptions{
		Mode:            "preinstalled",
		PrebuiltWDAPath: filepath.Join(wdaDerivedData, "Build", "Products", "Debug-iphonesimulator", "WebDriverAgentRunner-Runner.app"),
		LocalPort:       8100, MJPEGServerPort: 9100, LaunchTimeoutMs: 120_000,
	}
	if major := iosRuntimeMajor(destination.Lease.Runtime); major >= 17 && major < 27 {
		modes = append(modes, struct {
			name    string
			options *WDAOptions
		}{name: "preinstalled", options: preinstalled})
	}
	for _, mode := range modes {
		opened, err := service.open(ctx, &OpenRequest{
			SessionID: fmt.Sprintf("ios-wda-%s", mode.name), Destination: destination.Lease,
			Server: server.Server, BundleID: bundleID, WDA: mode.options,
		})
		if err != nil {
			t.Fatalf("open %s WDA mode: %v", mode.name, err)
		}
		result, err := service.run(ctx, &RunRequest{
			SessionID:       opened.Session.ID,
			Commands:        []interface{}{`expect(app.getByTestId("status")).toHaveText("Endly Mobile Runner", 30000)`},
			ActionTimeoutMs: 30_000, PollIntervalMs: 100,
		})
		if err != nil || len(result.Validations) != 1 || result.Validations[0].HasFailure() {
			t.Fatalf("probe %s WDA mode: response=%+v err=%v", mode.name, result, err)
		}
		if _, err := service.close(context.Background(), &CloseRequest{SessionID: opened.Session.ID}); err != nil {
			t.Fatalf("close %s WDA mode: %v", mode.name, err)
		}
	}
	if iosRuntimeMajor(destination.Lease.Runtime) >= 27 {
		_, err := service.open(ctx, &OpenRequest{
			SessionID: "ios-wda-preinstalled-unsupported", Destination: destination.Lease,
			Server: server.Server, BundleID: bundleID, WDA: preinstalled,
		})
		if err == nil || !strings.Contains(err.Error(), "preinstalled WDA is unavailable for iOS 27+") {
			t.Fatalf("expected explicit iOS 27 preinstalled compatibility error, got %v", err)
		}
	}

	cleanupErrors := cleanup.Close(context.Background())
	cleanupComplete = true
	if len(cleanupErrors) > 0 {
		t.Fatalf("WDA mode cleanup: %+v", cleanupErrors)
	}
}
