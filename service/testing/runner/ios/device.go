package ios

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func (s *service) deviceList(ctx *endly.Context, request *DeviceListRequest) (*DeviceListResponse, error) {
	devices, err := s.listPhysicalDevices(ctx.Background(), request.TimeoutMs)
	if err != nil {
		return nil, err
	}
	return &DeviceListResponse{Devices: devices}, nil
}

func (s *service) listPhysicalDevices(ctx context.Context, timeoutMs int) ([]PhysicalDevice, error) {
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	if timeoutMs <= 0 {
		timeoutMs = 30_000
	}
	listCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	result, err := s.runner.Run(listCtx, mobile.Command{Name: xcrun, Args: []string{
		"devicectl", "list", "devices", "--json-output", "-", "--omit-deprecated-fields-in-json",
		"--timeout", strconv.Itoa(max(1, timeoutMs/1000)),
	}})
	if err != nil {
		return nil, fmt.Errorf("list Apple physical devices: %s: %w", strings.TrimSpace(result.Stderr), err)
	}
	devices, err := parsePhysicalDevices(result.Stdout)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *service) deviceLease(ctx *endly.Context, request *DeviceLeaseRequest) (*DeviceLeaseResponse, error) {
	processLease, err := s.leaseStore.Acquire(ctx.Background(), "ios:device:"+request.UDID)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = s.leaseStore.Release(processLease)
		}
	}()
	devices, err := s.listPhysicalDevices(ctx.Background(), request.TimeoutMs)
	if err != nil {
		return nil, err
	}
	var selected *PhysicalDevice
	for index := range devices {
		device := &devices[index]
		if request.UDID == device.UDID {
			selected = device
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("Apple physical device %q is not connected", request.UDID)
	}
	if err := validatePhysicalDevice(*selected); err != nil {
		return nil, err
	}
	targetID := selected.UDID
	if targetID == "" {
		targetID = selected.Identifier
	}
	lease := DestinationLease{
		ID: uuid.NewString(), Fence: processLease.Fence, UDID: targetID, Name: selected.Name,
		Runtime: selected.OSVersion, Kind: "device", PreserveOnRelease: true, ProcessLease: processLease,
	}
	s.storeLease(lease)
	committed = true
	s.cleanupStack(ctx).Push("physical-device:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.releasePhysicalDevice(cleanupCtx, lease)
		return err
	})
	return &DeviceLeaseResponse{Lease: lease, Device: *selected}, nil
}

func (s *service) deviceRelease(ctx *endly.Context, request *DeviceReleaseRequest) (*DeviceReleaseResponse, error) {
	return s.releasePhysicalDevice(ctx.Background(), request.Lease)
}

func (s *service) releasePhysicalDevice(_ context.Context, lease DestinationLease) (*DeviceReleaseResponse, error) {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	s.mu.Unlock()
	if !ok {
		return &DeviceReleaseResponse{Warning: "physical-device lease already released or unknown"}, nil
	}
	if !stored.IsDevice() || stored.Fence != lease.Fence || stored.UDID != lease.UDID {
		return nil, fmt.Errorf("physical-device lease fence mismatch")
	}
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return nil, fmt.Errorf("physical-device persistent lease token mismatch")
		}
		if err := s.leaseStore.Validate(stored.ProcessLease); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
		return nil, err
	}
	return &DeviceReleaseResponse{Released: true}, nil
}

func parsePhysicalDevices(data string) ([]PhysicalDevice, error) {
	var envelope struct {
		Result struct {
			Devices []map[string]interface{} `json:"devices"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(data), &envelope); err != nil {
		return nil, fmt.Errorf("decode devicectl device list: %w", err)
	}
	result := make([]PhysicalDevice, 0, len(envelope.Result.Devices))
	for _, raw := range envelope.Result.Devices {
		device := PhysicalDevice{
			Identifier:    findDeviceString(raw, "identifier", "coreDeviceIdentifier"),
			UDID:          findDeviceString(raw, "udid", "uniqueDeviceIdentifier"),
			Name:          findDeviceString(raw, "name", "deviceName"),
			Platform:      findDeviceString(raw, "platform", "platformName"),
			OSVersion:     findDeviceString(raw, "osVersionNumber", "osVersion", "operatingSystemVersion"),
			State:         findDeviceString(raw, "visibilityClass", "availability", "deviceState"),
			Connection:    findDeviceString(raw, "transportType", "connectionType"),
			PairingState:  findDeviceString(raw, "pairingState", "pairState"),
			DeveloperMode: findDeviceString(raw, "developerModeStatus", "developerMode"),
		}
		if device.UDID == "" {
			device.UDID = device.Identifier
		}
		result = append(result, device)
	}
	return result, nil
}

func findDeviceString(value interface{}, keys ...string) string {
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[strings.ToLower(key)] = true
	}
	var visit func(interface{}) string
	visit = func(candidate interface{}) string {
		switch actual := candidate.(type) {
		case map[string]interface{}:
			for key, item := range actual {
				if wanted[strings.ToLower(key)] {
					switch scalar := item.(type) {
					case string:
						return scalar
					case float64:
						return strconv.FormatFloat(scalar, 'f', -1, 64)
					case bool:
						return strconv.FormatBool(scalar)
					}
				}
			}
			for _, item := range actual {
				if found := visit(item); found != "" {
					return found
				}
			}
		case []interface{}:
			for _, item := range actual {
				if found := visit(item); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return visit(value)
}

func validatePhysicalDevice(device PhysicalDevice) error {
	if device.Identifier == "" && device.UDID == "" {
		return fmt.Errorf("devicectl returned a device without an identifier")
	}
	platform := strings.ToLower(device.Platform)
	if platform != "" && !strings.Contains(platform, "ios") && !strings.Contains(platform, "iphone") && !strings.Contains(platform, "ipad") {
		return fmt.Errorf("device %q platform %q is not supported by the iOS runner", device.Name, device.Platform)
	}
	if pairing := strings.ToLower(device.PairingState); pairing != "" {
		unpaired := strings.Contains(pairing, "unpaired") || strings.Contains(pairing, "not paired") || strings.Contains(pairing, "notpaired")
		if unpaired || !strings.Contains(pairing, "paired") {
			return fmt.Errorf("device %q is not paired: %s", device.Name, device.PairingState)
		}
	}
	if mode := strings.ToLower(device.DeveloperMode); mode != "" {
		enabled := strings.Contains(mode, "enabled") || mode == "on" || mode == "true"
		if !enabled {
			return fmt.Errorf("device %q does not have Developer Mode enabled", device.Name)
		}
	}
	state := strings.ToLower(device.State)
	if strings.Contains(state, "unavailable") || strings.Contains(state, "provision") || strings.Contains(state, "disconnected") {
		return fmt.Errorf("device %q is not ready: %s", device.Name, device.State)
	}
	return nil
}

func parseDeviceProcessID(data string) int {
	var value interface{}
	if json.Unmarshal([]byte(data), &value) != nil {
		return 0
	}
	for _, key := range []string{"processIdentifier", "pid"} {
		if text := findDeviceString(value, key); text != "" {
			pid, _ := strconv.Atoi(strings.Split(text, ".")[0])
			if pid > 0 {
				return pid
			}
		}
	}
	return 0
}

func (s *service) installPhysicalApp(ctx *endly.Context, request *InstallRequest) (*InstallResponse, error) {
	if request.State == "freshInstall" {
		result, err := s.runDeviceCtl(ctx.Background(), []string{"device", "uninstall", "app", "--device", request.Destination.UDID, request.BundleID}, 120_000)
		if err != nil {
			detail := strings.ToLower(result.Stdout + result.Stderr + err.Error())
			if !strings.Contains(detail, "not installed") && !strings.Contains(detail, "not found") {
				return nil, fmt.Errorf("remove existing physical-device app %s: %w", request.BundleID, err)
			}
		}
	}
	result, err := s.runDeviceCtl(ctx.Background(), []string{
		"device", "install", "app", "--device", request.Destination.UDID,
		"--json-output", "-", request.App.HostPath,
	}, 180_000)
	if err != nil {
		return nil, fmt.Errorf("install physical-device app %s: %s: %w", request.BundleID, strings.TrimSpace(result.Stderr), err)
	}
	return &InstallResponse{Installed: true, BundleID: request.BundleID, AppPath: request.App.HostPath, State: request.State}, nil
}

func (s *service) uninstallPhysicalApp(ctx *endly.Context, request *UninstallRequest) (*UninstallResponse, error) {
	result, err := s.runDeviceCtl(ctx.Background(), []string{
		"device", "uninstall", "app", "--device", request.Destination.UDID, request.BundleID,
	}, 120_000)
	if err != nil {
		return nil, fmt.Errorf("uninstall physical-device app %s: %s: %w", request.BundleID, strings.TrimSpace(result.Stderr), err)
	}
	return &UninstallResponse{Removed: true, BundleID: request.BundleID}, nil
}

func (s *service) launchPhysicalApp(ctx *endly.Context, request *LaunchRequest) (*LaunchResponse, error) {
	environment, err := json.Marshal(request.Environment)
	if err != nil {
		return nil, err
	}
	args := []string{"device", "process", "launch", "--device", request.Destination.UDID, "--terminate-existing", "--json-output", "-"}
	if len(request.Environment) > 0 {
		args = append(args, "--environment-variables", string(environment))
	}
	args = append(args, request.BundleID)
	args = append(args, request.Arguments...)
	result, err := s.runDeviceCtl(ctx.Background(), args, 120_000)
	if err != nil {
		return nil, fmt.Errorf("launch physical-device app %s: %s: %w", request.BundleID, strings.TrimSpace(result.Stderr), err)
	}
	pid := parseDeviceProcessID(result.Stdout)
	if pid <= 0 {
		return nil, fmt.Errorf("devicectl launch succeeded without a process identifier")
	}
	return &LaunchResponse{PID: pid, Output: result.Stdout + result.Stderr}, nil
}

func (s *service) terminatePhysicalApp(ctx *endly.Context, request *TerminateRequest) (*TerminateResponse, error) {
	result, err := s.runDeviceCtl(ctx.Background(), []string{
		"device", "process", "terminate", "--device", request.Destination.UDID, "--pid", strconv.Itoa(request.PID),
	}, 120_000)
	if err != nil {
		return nil, fmt.Errorf("terminate physical-device process %d: %s: %w", request.PID, strings.TrimSpace(result.Stderr), err)
	}
	return &TerminateResponse{Terminated: true}, nil
}

func (s *service) runDeviceCtl(ctx context.Context, args []string, timeoutMs int) (mobile.Result, error) {
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return mobile.Result{}, err
	}
	commandCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	return s.runner.Run(commandCtx, mobile.Command{Name: xcrun, Args: append([]string{"devicectl"}, args...)})
}
