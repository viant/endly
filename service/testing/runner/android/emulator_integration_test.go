package android

import (
	"context"
	"os"
	"path/filepath"
	"sort"
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
	failureTestAPK := buildIntentionalFailureInstrumentation(t, sdkRoot, packageName)
	instrumentation, err := service.test(ctx, &TestRequest{
		Lease: started.Lease, AppAPKPath: APK.Path, TestAPKPath: failureTestAPK,
		TestPackage: "com.viant.endly.failuretest", Runner: ".FailureInstrumentation",
		GrantAll: true, TimeoutMs: 60_000,
	})
	if err != nil || instrumentation.Failed != 1 || len(instrumentation.Validations) != 1 || !instrumentation.Validations[0].HasFailure() {
		t.Fatalf("intentional instrumentation failure was not normalized: response=%+v err=%v", instrumentation, err)
	}
	if _, err := service.install(ctx, &InstallRequest{
		Lease: started.Lease, APKPath: APK.Path, Package: packageName, State: "freshInstall", GrantAll: true,
	}); err != nil {
		t.Fatal(err)
	}
	videoDirectory := t.TempDir()
	capture, err := service.captureStart(ctx, &CaptureStartRequest{
		Lease: started.Lease, LogPath: filepath.Join(t.TempDir(), "logcat.txt"), Clear: true,
		Video: true, VideoDirectory: videoDirectory, SegmentMs: 20_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ENDLY_ANDROID_APPIUM_INTEGRATION") == "1" {
		descriptorPath := filepath.Join(t.TempDir(), "android-appium-session.json")
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
			Package: packageName, Activity: activity, TestIDStrategy: "resourceId", DescriptorPath: descriptorPath,
		})
		if err != nil {
			t.Fatal(err)
		}
		run, err := service.run(ctx, &RunRequest{
			SessionID: opened.Session.ID,
			Commands: []interface{}{
				`expect(app.getByClass("android.widget.FrameLayout").first()).toBeVisible(60000)`,
				`textCount = app.getByClass("android.widget.TextView").count()`,
				`firstText = app.getByClass("android.widget.TextView").first().text()`,
				map[string]interface{}{
					"key":     "rootVisible",
					"locator": map[string]interface{}{"strategy": "class", "value": "android.widget.FrameLayout", "selection": "first"},
					"action":  map[string]interface{}{"name": "displayed"},
				},
				`contexts = device.contexts()`,
				`expect(device.context()).toHaveContext("NATIVE_APP", 10000)`,
				`expect(device.orientation()).toHaveOrientation("PORTRAIT", 10000)`,
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
		if count, ok := run.Data["textCount"].(int); !ok || count < 1 || run.Data["rootVisible"] != true {
			t.Fatalf("unexpected collection/typed data: %+v", run.Data)
		}
		evidence, err := service.artifact(ctx, &ArtifactRequest{
			SessionID: opened.Session.ID, Directory: t.TempDir(), Screenshot: true, PageSource: true,
		})
		if err != nil || len(evidence.Artifacts) != 2 {
			t.Fatalf("Appium evidence failed: response=%+v err=%v", evidence, err)
		}
		attachedService := newService(mobile.OSRunner{})
		attachedContext := endly.New().NewContext(nil)
		attached, err := attachedService.attach(attachedContext, &AttachRequest{DescriptorPath: descriptorPath})
		if err != nil {
			t.Fatalf("attach existing Appium session: %v", err)
		}
		attachedRun, err := attachedService.run(attachedContext, &RunRequest{
			SessionID: attached.Session.ID, Commands: []interface{}{`attachedContext = device.context()`},
			ActionTimeoutMs: 10_000, PollIntervalMs: 200,
		})
		if err != nil || attachedRun.Data["attachedContext"] != "NATIVE_APP" {
			t.Fatalf("attached Appium session failed: response=%+v err=%v", attachedRun, err)
		}
		if closed, err := attachedService.close(context.Background(), &CloseRequest{SessionID: attached.Session.ID}); err != nil || !strings.Contains(closed.Warning, "remains open") {
			t.Fatalf("safe attached-session close failed: response=%+v err=%v", closed, err)
		}
		failureResult, err := service.run(ctx, &RunRequest{
			SessionID:       opened.Session.ID,
			Commands:        []interface{}{`expect(device.orientation()).toHaveOrientation("LANDSCAPE", 100)`},
			ActionTimeoutMs: 1_000, PollIntervalMs: 50,
			FailureArtifacts: &mobile.FailureArtifactOptions{Directory: t.TempDir()},
		})
		if err != nil || len(failureResult.Failures) != 1 || !hasFailureKinds(failureResult.Failures[0], "screenshot", "pageSource", "logcat", "video", "failureManifest") {
			t.Fatalf("real automatic failure evidence incomplete: response=%+v err=%v", failureResult, err)
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
	if stopped, err := service.captureStop(context.Background(), &CaptureStopRequest{Capture: capture.Capture}); err != nil || !stopped.Stopped || !hasVideoEvidence(stopped.Artifacts) {
		t.Fatalf("capture stop failed: response=%+v err=%v", stopped, err)
	}
}

func buildIntentionalFailureInstrumentation(t *testing.T, sdkRoot, targetPackage string) string {
	t.Helper()
	buildToolsRoot := filepath.Join(sdkRoot, "build-tools")
	entries, err := os.ReadDir(buildToolsRoot)
	if err != nil {
		t.Fatal(err)
	}
	versions := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	buildTools := ""
	for _, version := range versions {
		candidate := filepath.Join(buildToolsRoot, version)
		if fixtureExecutable(filepath.Join(candidate, "aapt2")) && fixtureExecutable(filepath.Join(candidate, "d8")) && fixtureExecutable(filepath.Join(candidate, "zipalign")) && fixtureExecutable(filepath.Join(candidate, "apksigner")) {
			buildTools = candidate
			break
		}
	}
	if buildTools == "" {
		t.Fatal("aapt2, d8, zipalign, and apksigner are required in one Android build-tools version")
	}
	javaHome := os.Getenv("JAVA_HOME")
	if javaHome == "" {
		t.Fatal("JAVA_HOME is required to build the instrumentation failure fixture")
	}
	javac := filepath.Join(javaHome, "bin", "javac")
	jar := filepath.Join(javaHome, "bin", "jar")
	for _, executable := range []string{javac, jar} {
		if !fixtureExecutable(executable) {
			t.Fatalf("fixture build executable is missing: %s", executable)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	keystore := os.Getenv("ENDLY_ANDROID_TEST_KEYSTORE")
	if keystore == "" {
		keystore = filepath.Join(home, ".android", "debug.keystore")
	}
	if info, err := os.Stat(keystore); err != nil || info.IsDir() {
		t.Fatalf("debug keystore is required at %s: %v", keystore, err)
	}
	root := t.TempDir()
	classes := filepath.Join(root, "classes")
	dex := filepath.Join(root, "dex")
	if err := os.MkdirAll(classes, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dex, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestTemplate, err := os.ReadFile(filepath.Join("test", "fixture", "instrumentation", "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "AndroidManifest.xml")
	if err := os.WriteFile(manifest, []byte(strings.ReplaceAll(string(manifestTemplate), "${TARGET_PACKAGE}", targetPackage)), 0o600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join("test", "fixture", "instrumentation", "FailureInstrumentation.java")
	androidJar := filepath.Join(sdkRoot, "platforms", "android-35", "android.jar")
	commands := []mobile.Command{
		{Name: javac, Args: []string{"-source", "8", "-target", "8", "-classpath", androidJar, "-d", classes, source}},
		{Name: filepath.Join(buildTools, "d8"), Args: []string{"--min-api", "26", "--output", dex, filepath.Join(classes, "com", "viant", "endly", "failuretest", "FailureInstrumentation.class")}},
	}
	for _, command := range commands {
		if result, err := (mobile.OSRunner{}).Run(context.Background(), command); err != nil {
			t.Fatalf("build failure instrumentation with %s: %v\n%s%s", command.Name, err, result.Stdout, result.Stderr)
		}
	}
	unsignedAPK := filepath.Join(root, "failure-unsigned.apk")
	if result, err := (mobile.OSRunner{}).Run(context.Background(), mobile.Command{
		Name: filepath.Join(buildTools, "aapt2"), Args: []string{"link", "-o", unsignedAPK, "--manifest", manifest, "-I", androidJar},
	}); err != nil {
		t.Fatalf("link failure instrumentation: %v\n%s%s", err, result.Stdout, result.Stderr)
	}
	if result, err := (mobile.OSRunner{}).Run(context.Background(), mobile.Command{
		Name: jar, Args: []string{"uf", unsignedAPK, "-C", dex, "classes.dex"},
	}); err != nil {
		t.Fatalf("package failure instrumentation dex: %v\n%s%s", err, result.Stdout, result.Stderr)
	}
	alignedAPK := filepath.Join(root, "failure-aligned.apk")
	if result, err := (mobile.OSRunner{}).Run(context.Background(), mobile.Command{
		Name: filepath.Join(buildTools, "zipalign"), Args: []string{"-f", "4", unsignedAPK, alignedAPK},
	}); err != nil {
		t.Fatalf("align failure instrumentation: %v\n%s%s", err, result.Stdout, result.Stderr)
	}
	signedAPK := filepath.Join(root, "failure-test.apk")
	if result, err := (mobile.OSRunner{}).Run(context.Background(), mobile.Command{
		Name: filepath.Join(buildTools, "apksigner"), Args: []string{
			"sign", "--ks", keystore, "--ks-key-alias", "androiddebugkey",
			"--ks-pass", "pass:android", "--key-pass", "pass:android", "--out", signedAPK, alignedAPK,
		},
	}); err != nil {
		t.Fatalf("sign failure instrumentation: %v\n%s%s", err, result.Stdout, result.Stderr)
	}
	return signedAPK
}

func fixtureExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func hasFailureKinds(failure *mobile.FailureEvidence, kinds ...string) bool {
	if failure == nil || len(failure.Errors) > 0 {
		return false
	}
	found := map[string]bool{}
	for _, artifact := range failure.Artifacts {
		if artifact != nil && (artifact.Kind != "video" || artifact.Size > 0) {
			found[artifact.Kind] = true
		}
	}
	for _, kind := range kinds {
		if !found[kind] {
			return false
		}
	}
	return true
}

func hasVideoEvidence(artifacts []*mobile.Evidence) bool {
	for _, artifact := range artifacts {
		if artifact != nil && artifact.Kind == "video" && artifact.Size > 0 {
			return true
		}
	}
	return false
}
