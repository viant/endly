//go:build darwin

package ios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

const physicalDevicesJSON = `{
  "info":{"outcome":"success","jsonVersion":5},
  "result":{"devices":[{
    "identifier":"CORE-1",
    "properties":{
      "hardware":{"udid":"UDID-1"},
      "device":{"name":"Test iPhone","operatingSystemVersion":"iOS 18.5","developerModeStatus":"enabled"},
      "connection":{"pairingState":"paired","transportType":"wired"},
      "state":{"visibilityClass":"default"},
      "platform":"iOS"
    }
  }]}
}`

func TestPhysicalDeviceLeaseDeploymentAndProcessLifecycle(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		joined := strings.Join(command.Args, " ")
		switch {
		case strings.Contains(joined, "devicectl list devices"):
			return mobile.Result{Stdout: physicalDevicesJSON}, nil
		case strings.Contains(joined, "devicectl device process launch"):
			return mobile.Result{Stdout: `{"result":{"process":{"processIdentifier":4321}}}`}, nil
		default:
			return mobile.Result{Stdout: `{"info":{"outcome":"success"}}`}, nil
		}
	}}
	service := newService(runner)
	service.leaseStore = mobile.NewLeaseStore(t.TempDir())
	ctx := endly.New().NewContext(nil)
	listed, err := service.deviceList(ctx, &DeviceListRequest{TimeoutMs: 1000})
	if err != nil || len(listed.Devices) != 1 || listed.Devices[0].UDID != "UDID-1" {
		t.Fatalf("device list=%+v err=%v", listed, err)
	}
	leased, err := service.deviceLease(ctx, &DeviceLeaseRequest{UDID: "UDID-1", TimeoutMs: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if !leased.Lease.IsDevice() || leased.Lease.ProcessLease == nil || leased.Lease.Runtime != "iOS 18.5" {
		t.Fatalf("unexpected physical lease: %+v", leased)
	}
	appPath := filepath.Join(t.TempDir(), "Fixture.app")
	if err := os.MkdirAll(appPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := service.install(ctx, &InstallRequest{
		Destination: leased.Lease, App: Artifact{Kind: "deviceApp", HostPath: appPath},
		BundleID: "com.example.fixture", State: "preserve",
	}); err != nil {
		t.Fatal(err)
	}
	launched, err := service.launch(ctx, &LaunchRequest{
		Destination: leased.Lease, BundleID: "com.example.fixture",
		Arguments: []string{"--uitesting"}, Environment: map[string]string{"MODE": "test"},
	})
	if err != nil || launched.PID != 4321 {
		t.Fatalf("launch=%+v err=%v", launched, err)
	}
	if _, err := service.terminate(ctx, &TerminateRequest{Destination: leased.Lease, PID: launched.PID}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.uninstall(ctx, &UninstallRequest{Destination: leased.Lease, BundleID: "com.example.fixture"}); err != nil {
		t.Fatal(err)
	}
	released, err := service.deviceRelease(ctx, &DeviceReleaseRequest{Lease: leased.Lease})
	if err != nil || !released.Released {
		t.Fatalf("release=%+v err=%v", released, err)
	}
	if _, err := os.Stat(leased.Lease.ProcessLease.Path); !os.IsNotExist(err) {
		t.Fatalf("physical lease remains: %v", err)
	}
	commands := recordedCommands(runner)
	for _, expected := range []string{
		"devicectl device install app --device UDID-1 --json-output - " + appPath,
		`devicectl device process launch --device UDID-1 --terminate-existing --json-output - --environment-variables {"MODE":"test"} com.example.fixture --uitesting`,
		"devicectl device process terminate --device UDID-1 --pid 4321",
		"devicectl device uninstall app --device UDID-1 com.example.fixture",
	} {
		if !strings.Contains(commands, expected) {
			t.Errorf("missing %q in:\n%s", expected, commands)
		}
	}
}

func TestPhysicalDeviceEligibilityAndBuildDestination(t *testing.T) {
	if err := validatePhysicalDevice(PhysicalDevice{Identifier: "watch", Platform: "watchOS"}); err == nil {
		t.Fatal("expected non-iOS device rejection")
	}
	if err := validatePhysicalDevice(PhysicalDevice{Identifier: "phone", Platform: "iOS", PairingState: "unpaired"}); err == nil {
		t.Fatal("expected pairing rejection")
	}
	if err := validatePhysicalDevice(PhysicalDevice{Identifier: "phone", Platform: "iOS", DeveloperMode: "disabled"}); err == nil {
		t.Fatal("expected Developer Mode rejection")
	}
	args := strings.Join(iosBuildArgs(&BuildRequest{
		ProjectPath: "/tmp/Fixture.xcodeproj", Scheme: "Fixture", Configuration: "Debug", Mode: "build",
		Destination: DestinationLease{Kind: "device", UDID: "UDID-1"}, DerivedDataPath: "/tmp/DerivedData",
	}), " ")
	if !strings.Contains(args, "-destination platform=iOS,id=UDID-1") {
		t.Fatalf("physical build destination missing: %s", args)
	}
}

func TestXCTestUsesPhysicalDeviceDestination(t *testing.T) {
	runner := &fakeRunner{runHook: func(command mobile.Command) (mobile.Result, error) {
		if strings.Contains(strings.Join(command.Args, " "), "xcresulttool get test-results summary") {
			return mobile.Result{Stdout: `{"title":"DeviceTests","result":"Passed","totalTestCount":1,"passedTests":1,"failedTests":0}`}, nil
		}
		return mobile.Result{}, nil
	}}
	service := newService(runner)
	lease := DestinationLease{ID: "device-lease", Fence: 1, Kind: "device", UDID: "UDID-1"}
	service.storeLease(lease)
	response, err := service.test(endly.New().NewContext(nil), &TestRequest{
		Destination: lease, ProjectPath: "/tmp/Fixture.xcodeproj", Scheme: "Fixture",
		Mode: "scheme", ResultBundlePath: "/tmp/device.xcresult", CollectDiagnostics: "never", TimeoutMs: 1000,
	})
	if err != nil || response.Summary.PassedTests != 1 {
		t.Fatalf("physical XCTest response=%+v err=%v", response, err)
	}
	if !strings.Contains(recordedCommands(runner), "-destination platform=iOS,id=UDID-1") {
		t.Fatalf("physical XCTest destination missing:\n%s", recordedCommands(runner))
	}
}

func TestParseDeviceProcessID(t *testing.T) {
	if pid := parseDeviceProcessID(`{"result":{"process":{"processIdentifier":9876}}}`); pid != 9876 {
		t.Fatalf("pid=%d", pid)
	}
}

func TestPhysicalDeviceListIntegration(t *testing.T) {
	if os.Getenv("ENDLY_IOS_DEVICE_LIST_INTEGRATION") != "1" {
		t.Skip("set ENDLY_IOS_DEVICE_LIST_INTEGRATION=1 to query CoreDevice")
	}
	service := newService(mobile.OSRunner{})
	response, err := service.deviceList(endly.New().NewContext(nil), &DeviceListRequest{TimeoutMs: 30_000})
	if err != nil {
		t.Fatal(err)
	}
	if response.Devices == nil {
		t.Fatal("device list must be an empty slice rather than nil when no device is connected")
	}
}
