package android

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

func TestAndroidEmulatorIntegration(t *testing.T) {
	if os.Getenv("ENDLY_ANDROID_EMULATOR_INTEGRATION") != "1" {
		t.Skip("set ENDLY_ANDROID_EMULATOR_INTEGRATION=1 to run the real Android emulator integration")
	}
	sdkRoot := os.Getenv("ANDROID_SDK_ROOT")
	avd := os.Getenv("ENDLY_ANDROID_TEST_AVD")
	projectDir := os.Getenv("ENDLY_ANDROID_TEST_PROJECT")
	packageName := os.Getenv("ENDLY_ANDROID_TEST_PACKAGE")
	if sdkRoot == "" || avd == "" || projectDir == "" || packageName == "" {
		t.Fatal("ANDROID_SDK_ROOT, ENDLY_ANDROID_TEST_AVD, ENDLY_ANDROID_TEST_PROJECT, and ENDLY_ANDROID_TEST_PACKAGE are required")
	}
	module := os.Getenv("ENDLY_ANDROID_TEST_MODULE")
	if module == "" {
		module = "app"
	}
	variant := os.Getenv("ENDLY_ANDROID_TEST_VARIANT")
	if variant == "" {
		variant = "debug"
	}
	activity := os.Getenv("ENDLY_ANDROID_TEST_ACTIVITY")

	service := newService(mobile.OSRunner{})
	ctx := endly.New().NewContext(nil)
	doctorRequest := &DoctorRequest{AndroidSDKRoot: sdkRoot, Required: []string{"adb", "emulator"}}
	if err := doctorRequest.Init(); err != nil {
		t.Fatal(err)
	}
	doctor, err := service.doctor(ctx, doctorRequest)
	if err != nil || !doctor.Ready {
		t.Fatalf("Android emulator host is not ready: response=%+v err=%v", doctor, err)
	}
	started, err := service.deviceStart(ctx, &DeviceStartRequest{
		AndroidSDKRoot: sdkRoot, AVD: avd, WipeData: true, NoWindow: true,
		NoSnapshot: true, BootTimeoutMs: 180_000, PollIntervalMs: 1_000,
		LogPath: filepath.Join(t.TempDir(), "emulator.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if errors := service.cleanupStack(ctx).Close(context.Background()); len(errors) > 0 {
			t.Errorf("cleanup errors: %+v", errors)
		}
	}()
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
		if built == nil {
			t.Fatal(err)
		}
		t.Fatalf("%v\nstdout:\n%s\nstderr:\n%s", err, built.Stdout, built.Stderr)
	}
	var APK BuildArtifact
	for _, candidate := range built.Artifacts {
		if candidate.Kind == "appAPK" {
			APK = candidate
			break
		}
	}
	if APK.Path == "" {
		t.Fatalf("no application APK in artifacts: %+v", built.Artifacts)
	}
	if _, err := service.install(ctx, &InstallRequest{
		Lease: started.Lease, APKPath: APK.Path, Package: packageName, State: "freshInstall", GrantAll: true,
	}); err != nil {
		t.Fatal(err)
	}
	capture, err := service.captureStart(ctx, &CaptureStartRequest{
		Lease: started.Lease, LogPath: filepath.Join(t.TempDir(), "logcat.txt"), Clear: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ENDLY_ANDROID_APPIUM_INTEGRATION") == "1" {
		executable := os.Getenv("ENDLY_ANDROID_APPIUM_EXECUTABLE")
		appiumHome := os.Getenv("ENDLY_ANDROID_APPIUM_HOME")
		server, err := service.serverStart(ctx, &ServerStartRequest{
			Lease: started.Lease, Mode: "managed", Executable: executable, AppiumHome: appiumHome,
			Address: "127.0.0.1", Port: 4723, LogPath: filepath.Join(t.TempDir(), "appium.log"),
		})
		if err != nil {
			t.Fatal(err)
		}
		opened, err := service.open(ctx, &OpenRequest{
			SessionID: "android-fixture", Lease: started.Lease, Server: server.Server,
			Package: packageName, Activity: activity, TestIDStrategy: "resourceId",
		})
		if err != nil {
			t.Fatal(err)
		}
		run, err := service.run(ctx, &RunRequest{
			SessionID: opened.Session.ID,
			Commands: []string{
				`expect(app.getByClass("android.widget.FrameLayout")).toBeVisible(60000)`,
			},
			ActionTimeoutMs: 60_000,
			PollIntervalMs:  200,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, validation := range run.Validations {
			if validation.HasFailure() {
				t.Fatalf("Android DSL assertion failed: %s", validation.Report())
			}
		}
		evidence, err := service.artifact(ctx, &ArtifactRequest{
			SessionID: opened.Session.ID, Directory: t.TempDir(), Screenshot: true, PageSource: true,
		})
		if err != nil || len(evidence.Artifacts) != 2 {
			t.Fatalf("Appium evidence failed: response=%+v err=%v", evidence, err)
		}
	} else {
		if _, err := service.launch(ctx, &LaunchRequest{Lease: started.Lease, Package: packageName, Activity: activity}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		evidence, err := service.artifact(ctx, &ArtifactRequest{
			Lease: &started.Lease, Directory: t.TempDir(), Screenshot: true,
		})
		if err != nil || len(evidence.Artifacts) != 1 || evidence.Artifacts[0].Size == 0 {
			t.Fatalf("emulator screenshot failed: response=%+v err=%v", evidence, err)
		}
	}
	if _, err := service.terminate(ctx, &TerminateRequest{Lease: started.Lease, Package: packageName}); err != nil {
		t.Fatal(err)
	}
	if stopped, err := service.captureStop(context.Background(), &CaptureStopRequest{Capture: capture.Capture}); err != nil || !stopped.Stopped {
		t.Fatalf("capture stop failed: response=%+v err=%v", stopped, err)
	}
}
