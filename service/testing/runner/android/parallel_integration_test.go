package android

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

type androidStressLane struct {
	index   int
	avd     string
	root    string
	device  *DeviceStartResponse
	server  *ServerStartResponse
	session *OpenResponse
}

func TestAndroidParallelEmulatorStress(t *testing.T) {
	if os.Getenv("ENDLY_ANDROID_PARALLEL_INTEGRATION") != "1" {
		t.Skip("set ENDLY_ANDROID_PARALLEL_INTEGRATION=1 to run two concurrent Android emulator lanes")
	}
	sdkRoot := os.Getenv("ANDROID_SDK_ROOT")
	projectDir := os.Getenv("ENDLY_ANDROID_TEST_PROJECT")
	packageName := os.Getenv("ENDLY_ANDROID_TEST_PACKAGE")
	appiumExecutable := os.Getenv("ENDLY_ANDROID_APPIUM_EXECUTABLE")
	appiumHome := os.Getenv("ENDLY_ANDROID_APPIUM_HOME")
	avds := splitNonEmpty(os.Getenv("ENDLY_ANDROID_PARALLEL_AVDS"))
	if sdkRoot == "" || projectDir == "" || packageName == "" || appiumExecutable == "" || appiumHome == "" || len(avds) != 2 {
		t.Fatal("ANDROID_SDK_ROOT, ENDLY_ANDROID_TEST_PROJECT, ENDLY_ANDROID_TEST_PACKAGE, Appium executable/home, and exactly two ENDLY_ANDROID_PARALLEL_AVDS are required")
	}
	activity := os.Getenv("ENDLY_ANDROID_TEST_ACTIVITY")
	service := newService(mobile.OSRunner{})
	ctx := endly.New().NewContext(nil)
	cleanup := service.cleanupStack(ctx)
	cleanupComplete := false
	defer func() {
		if !cleanupComplete {
			if cleanupErrors := cleanup.Close(context.Background()); len(cleanupErrors) > 0 {
				t.Errorf("parallel Android fallback cleanup: %+v", cleanupErrors)
			}
		}
	}()

	lanes := []*androidStressLane{
		{index: 0, avd: avds[0], root: t.TempDir()},
		{index: 1, avd: avds[1], root: t.TempDir()},
	}
	for _, lane := range lanes {
		animations := false
		started, err := service.deviceStart(ctx, &DeviceStartRequest{
			AndroidSDKRoot: sdkRoot, AVD: lane.avd, Port: 5554 + lane.index*2,
			WipeData: true, NoWindow: true, NoSnapshot: true, Animations: &animations,
			BootTimeoutMs: 240_000, PollIntervalMs: 1_000,
			LogPath: filepath.Join(lane.root, "emulator.log"),
		})
		lane.device = started
		if err != nil {
			t.Fatalf("start Android parallel lane %d: %v", lane.index, err)
		}
	}

	variant := os.Getenv("ENDLY_ANDROID_TEST_VARIANT")
	if variant == "" {
		variant = "debug"
	}
	module := os.Getenv("ENDLY_ANDROID_TEST_MODULE")
	if module == "" {
		module = "app"
	}
	variantTask := strings.ToUpper(variant[:1]) + variant[1:]
	gradleArgs := []string{}
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		gradleArgs = append(gradleArgs, "-Dorg.gradle.java.home="+javaHome)
	}
	built, err := service.build(ctx, &BuildRequest{
		ProjectDir: projectDir, Module: module, Variant: variant,
		Tasks: []string{"assemble" + variantTask}, GradleArgs: gradleArgs, TimeoutMs: 20 * 60 * 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	APKPath := ""
	for _, artifact := range built.Artifacts {
		if artifact.Kind == "appAPK" {
			APKPath = artifact.Path
			break
		}
	}
	if APKPath == "" {
		t.Fatalf("no app APK in build artifacts: %+v", built.Artifacts)
	}

	if err := runAndroidStressLanes(lanes, func(lane *androidStressLane) error {
		_, err := service.install(ctx, &InstallRequest{
			Lease: lane.device.Lease, APKPath: APKPath, Package: packageName, State: "freshInstall", GrantAll: true,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runAndroidStressLanes(lanes, func(lane *androidStressLane) error {
		started, err := service.serverStart(ctx, &ServerStartRequest{
			Lease: lane.device.Lease, Mode: "managed", Executable: appiumExecutable, AppiumHome: appiumHome,
			Address: "127.0.0.1", Port: 4723 + lane.index*2,
			LogPath: filepath.Join(lane.root, "appium.log"), StartupTimeoutMs: 45_000,
		})
		lane.server = started
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runAndroidStressLanes(lanes, func(lane *androidStressLane) error {
		opened, err := service.open(ctx, &OpenRequest{
			SessionID: fmt.Sprintf("android-parallel-%d", lane.index), Lease: lane.device.Lease,
			Server: lane.server.Server, Package: packageName, Activity: activity, TestIDStrategy: "resourceId",
			Capabilities: map[string]interface{}{
				"appium:systemPort":      8200 + lane.index,
				"appium:mjpegServerPort": 9200 + lane.index,
			},
		})
		lane.session = opened
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := runAndroidStressLanes(lanes, func(lane *androidStressLane) error {
		commands := make([]interface{}, 0, 12)
		for iteration := 0; iteration < 6; iteration++ {
			commands = append(commands,
				`expect(app.getByClass("android.widget.FrameLayout").first()).toBeVisible(30000)`,
				`expect(device.orientation()).toHaveOrientation("PORTRAIT", 5000)`,
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
		t.Fatalf("parallel Android cleanup: %+v", cleanupErrors)
	}
	for _, lane := range lanes {
		if lane.device == nil || lane.server == nil || lane.session == nil {
			t.Fatalf("incomplete lane: %+v", lane)
		}
		for _, path := range []string{lane.device.Lease.ProcessLease.Path, lane.server.Server.ProcessLease.Path} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("lane %d lease remains at %s: %v", lane.index, path, err)
			}
		}
	}
}

func runAndroidStressLanes(lanes []*androidStressLane, operation func(*androidStressLane) error) error {
	var wait sync.WaitGroup
	errorsByLane := make(chan error, len(lanes))
	for _, lane := range lanes {
		lane := lane
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := operation(lane); err != nil {
				errorsByLane <- fmt.Errorf("Android parallel lane %d: %w", lane.index, err)
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

func splitNonEmpty(value string) []string {
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
