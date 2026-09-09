//go:build darwin

package ios

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

type iosStressLane struct {
	index       int
	deviceType  string
	root        string
	destination *SimulatorStartResponse
	server      *ServerStartResponse
	session     *OpenResponse
}

func TestIOSParallelSimulatorStress(t *testing.T) {
	if os.Getenv("ENDLY_IOS_PARALLEL_INTEGRATION") != "1" {
		t.Skip("set ENDLY_IOS_PARALLEL_INTEGRATION=1 to run two concurrent iOS Simulator lanes")
	}
	appiumExecutable := os.Getenv("ENDLY_IOS_APPIUM_EXECUTABLE")
	appiumHome := os.Getenv("ENDLY_IOS_APPIUM_HOME")
	if appiumExecutable == "" || appiumHome == "" {
		t.Fatal("ENDLY_IOS_APPIUM_EXECUTABLE and ENDLY_IOS_APPIUM_HOME are required")
	}
	deviceTypes := splitIOSNonEmpty(os.Getenv("ENDLY_IOS_PARALLEL_DEVICE_TYPES"))
	if len(deviceTypes) == 0 {
		deviceTypes = []string{
			"com.apple.CoreSimulator.SimDeviceType.iPhone-17-Pro",
			"com.apple.CoreSimulator.SimDeviceType.iPhone-17-Pro-Max",
		}
	}
	if len(deviceTypes) != 2 || deviceTypes[0] == deviceTypes[1] {
		t.Fatal("ENDLY_IOS_PARALLEL_DEVICE_TYPES must contain exactly two distinct device type identifiers")
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

	service := newService(mobile.OSRunner{})
	ctx := endly.New().NewContext(nil)
	cleanup := service.cleanupStack(ctx)
	cleanupComplete := false
	defer func() {
		if !cleanupComplete {
			if cleanupErrors := cleanup.Close(context.Background()); len(cleanupErrors) > 0 {
				t.Errorf("parallel iOS fallback cleanup: %+v", cleanupErrors)
			}
		}
	}()
	doctor, err := service.doctor(ctx, &DoctorRequest{Required: []string{"xcodebuild", "simulator-runtime"}})
	if err != nil || !doctor.Ready || len(doctor.Runtimes) == 0 {
		t.Fatalf("iOS host not ready: response=%+v err=%v", doctor, err)
	}
	runtimeID := doctor.Runtimes[len(doctor.Runtimes)-1]
	lanes := []*iosStressLane{
		{index: 0, deviceType: deviceTypes[0], root: t.TempDir()},
		{index: 1, deviceType: deviceTypes[1], root: t.TempDir()},
	}
	if err := runIOSStressLanes(lanes, func(lane *iosStressLane) error {
		started, err := service.simulatorStart(ctx, &SimulatorStartRequest{
			DeviceType: lane.deviceType, Runtime: runtimeID, BootTimeoutMs: 240_000,
		})
		lane.destination = started
		return err
	}); err != nil {
		t.Fatal(err)
	}

	derivedData := filepath.Join(t.TempDir(), "DerivedData")
	built, err := service.build(ctx, &BuildRequest{
		ProjectPath: projectPath, Scheme: scheme, Configuration: "Debug",
		Destination: lanes[0].destination.Lease, DerivedDataPath: derivedData, Mode: "build", TimeoutMs: 20 * 60 * 1000,
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
	if err := runIOSStressLanes(lanes, func(lane *iosStressLane) error {
		_, err := service.install(ctx, &InstallRequest{
			Destination: lane.destination.Lease, App: app, BundleID: bundleID, State: "freshInstall",
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runIOSStressLanes(lanes, func(lane *iosStressLane) error {
		started, err := service.serverStart(ctx, &ServerStartRequest{
			Destination: lane.destination.Lease, Mode: "managed", Executable: appiumExecutable, AppiumHome: appiumHome,
			Address: "127.0.0.1", Port: 4723 + lane.index*2,
			LogPath: filepath.Join(lane.root, "appium.log"), StartupTimeoutMs: 60_000,
		})
		lane.server = started
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runIOSStressLanes(lanes, func(lane *iosStressLane) error {
		opened, err := service.open(ctx, &OpenRequest{
			SessionID: fmt.Sprintf("ios-parallel-%d", lane.index), Destination: lane.destination.Lease,
			Server: lane.server.Server, BundleID: bundleID,
			Capabilities: map[string]interface{}{
				"appium:wdaLocalPort":      8100 + lane.index,
				"appium:mjpegServerPort":   9100 + lane.index,
				"appium:derivedDataPath":   filepath.Join(lane.root, "WDA"),
				"appium:useNewWDA":         true,
				"appium:wdaStartupRetries": 2,
			},
		})
		lane.session = opened
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runIOSStressLanes(lanes, func(lane *iosStressLane) error {
		commands := make([]interface{}, 0, 12)
		for iteration := 1; iteration <= 6; iteration++ {
			commands = append(commands,
				`app.getByTestId("increment").tap()`,
				fmt.Sprintf(`expect(app.getByTestId("count")).toHaveText("Count: %d", 10000)`, iteration),
			)
		}
		result, err := service.run(ctx, &RunRequest{
			SessionID: lane.session.Session.ID, Commands: commands, ActionTimeoutMs: 30_000, PollIntervalMs: 100,
		})
		if err != nil {
			return err
		}
		for _, validation := range result.Validations {
			if validation.HasFailure() {
				return fmt.Errorf("lane %d validation: %s", lane.index, validation.Report())
			}
		}
		evidence, err := service.artifact(ctx, &ArtifactRequest{
			SessionID: lane.session.Session.ID, Directory: filepath.Join(lane.root, "evidence"), Screenshot: true, PageSource: true,
		})
		if err != nil || len(evidence.Artifacts) != 2 {
			return fmt.Errorf("lane %d evidence=%+v: %w", lane.index, evidence, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cleanupErrors := cleanup.Close(context.Background())
	cleanupComplete = true
	if len(cleanupErrors) > 0 {
		t.Fatalf("parallel iOS cleanup: %+v", cleanupErrors)
	}
	for _, lane := range lanes {
		if lane.destination == nil || lane.server == nil || lane.session == nil {
			t.Fatalf("incomplete lane: %+v", lane)
		}
		for _, path := range []string{lane.destination.Lease.ProcessLease.Path, lane.server.Server.ProcessLease.Path} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("lane %d lease remains at %s: %v", lane.index, path, err)
			}
		}
	}
}

func runIOSStressLanes(lanes []*iosStressLane, operation func(*iosStressLane) error) error {
	var wait sync.WaitGroup
	errorsByLane := make(chan error, len(lanes))
	for _, lane := range lanes {
		lane := lane
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := operation(lane); err != nil {
				errorsByLane <- fmt.Errorf("iOS parallel lane %d: %w", lane.index, err)
			}
		}()
	}
	wait.Wait()
	close(errorsByLane)
	result := []error{}
	for err := range errorsByLane {
		result = append(result, err)
	}
	return errors.Join(result...)
}

func splitIOSNonEmpty(value string) []string {
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
