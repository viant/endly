package ios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/endly"
)

func TestWDAOptionsMapOwnedCapabilities(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	root := t.TempDir()
	derivedData := filepath.Join(root, "DerivedData")
	prebuiltApp := filepath.Join(root, "WebDriverAgentRunner-Runner.app")
	for _, directory := range []string{derivedData, prebuiltApp} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	xcodeConfig := filepath.Join(root, "wda.xcconfig")
	keychain := filepath.Join(root, "wda.p12")
	password := filepath.Join(root, "wda.pass")
	for path, data := range map[string]string{xcodeConfig: "DEVELOPMENT_TEAM=TEAM", keychain: "p12", password: "secret\n"} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	useNew := false
	managed, err := wdaCapabilities(ctx, &WDAOptions{
		Mode: "managed", DerivedDataPath: derivedData, XcodeOrgID: "TEAM", XcodeSigningID: "Apple Development",
		XcodeConfigFile: xcodeConfig, KeychainPath: keychain, KeychainPasswordFile: password,
		LocalPort: 8101, MJPEGServerPort: 9101, UseNewWDA: &useNew, PrebuildWDA: true,
		LaunchTimeoutMs: 20_000, ConnectionTimeoutMs: 30_000, StartupRetries: 2, StartupRetryMs: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]interface{}{
		"appium:derivedDataPath": derivedData, "appium:xcodeOrgId": "TEAM",
		"appium:xcodeSigningId": "Apple Development", "appium:xcodeConfigFile": xcodeConfig,
		"appium:keychainPath": keychain, "appium:keychainPassword": "secret",
		"appium:wdaLocalPort": 8101, "appium:mjpegServerPort": 9101,
		"appium:useNewWDA": false, "appium:prebuildWDA": true,
		"appium:wdaLaunchTimeout": 20_000, "appium:wdaConnectionTimeout": 30_000,
		"appium:wdaStartupRetries": 2, "appium:wdaStartupRetryInterval": 500,
	} {
		if actual, ok := managed[key]; !ok || actual != expected {
			t.Errorf("capability %s=%v, want %v", key, actual, expected)
		}
	}
	prebuilt, err := wdaCapabilities(ctx, &WDAOptions{Mode: "prebuilt", DerivedDataPath: derivedData})
	if err != nil || prebuilt["appium:usePrebuiltWDA"] != true {
		t.Fatalf("prebuilt capabilities=%v err=%v", prebuilt, err)
	}
	emptySuffix := ""
	preinstalled, err := wdaCapabilities(ctx, &WDAOptions{
		Mode: "preinstalled", PrebuiltWDAPath: prebuiltApp,
		UpdatedBundleID: "io.appium.wda", UpdatedBundleIDSuffix: &emptySuffix,
	})
	if err != nil || preinstalled["appium:usePreinstalledWDA"] != true || preinstalled["appium:updatedWDABundleIdSuffix"] != "" {
		t.Fatalf("preinstalled capabilities=%v err=%v", preinstalled, err)
	}
	external, err := wdaCapabilities(ctx, &WDAOptions{Mode: "external", WebDriverAgentURL: "http://127.0.0.1:8100"})
	if err != nil || external["appium:webDriverAgentUrl"] != "http://127.0.0.1:8100" {
		t.Fatalf("external capabilities=%v err=%v", external, err)
	}
}

func TestWDAOptionsRejectConflictsAndUnprotectedSecrets(t *testing.T) {
	for _, options := range []*WDAOptions{
		{Mode: "unknown"},
		{Mode: "prebuilt"},
		{Mode: "external", WebDriverAgentURL: "file:///tmp/wda"},
		{Mode: "managed", XcodeOrgID: "TEAM"},
		{Mode: "managed", KeychainPath: "/tmp/keychain"},
	} {
		if err := options.Validate(); err == nil {
			t.Fatalf("expected invalid WDA options: %+v", options)
		}
	}
	for _, key := range []string{"appium:usePrebuiltWDA", "usePreinstalledWDA", "appium:webDriverAgentUrl", "keychainPassword", "appium:wdaLocalPort"} {
		if !isProtectedIOSCapability(key) {
			t.Errorf("WDA capability %q was not protected", key)
		}
	}
}

func TestPreinstalledWDACompatibility(t *testing.T) {
	options := &WDAOptions{Mode: "preinstalled"}
	if err := validateWDACompatibility(options, DestinationLease{Runtime: "com.apple.CoreSimulator.SimRuntime.iOS-16-4"}); err == nil {
		t.Fatal("expected iOS 16 preinstalled WDA rejection")
	}
	if err := validateWDACompatibility(options, DestinationLease{Runtime: "com.apple.CoreSimulator.SimRuntime.iOS-18-5"}); err != nil {
		t.Fatal(err)
	}
	if err := validateWDACompatibility(options, DestinationLease{Runtime: "com.apple.CoreSimulator.SimRuntime.iOS-27-0"}); err == nil {
		t.Fatal("expected iOS 27 preinstalled WDA rejection")
	}
}
