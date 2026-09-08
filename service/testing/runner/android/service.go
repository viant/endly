package android

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viant/afs"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

const ServiceID = "android"

type service struct {
	*endly.AbstractService
	runner   mobile.Runner
	mu       sync.Mutex
	leases   map[string]DeviceLease
	sessions map[string]*androidSession
	servers  map[string]*mobile.AppiumServer
	captures map[string]*mobile.LoggedProcess
	fs       afs.Service
}

type androidSession struct {
	handle         SessionHandle
	appium         *mobile.AppiumSession
	testIDStrategy string
	mu             sync.Mutex
}

type contextCleanup struct{ stack *mobile.CleanupStack }

var cleanupKey = (*contextCleanup)(nil)

func New() endly.Service {
	return newService(mobile.OSRunner{})
}

func newService(runner mobile.Runner) *service {
	result := &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		runner:          runner,
		leases:          map[string]DeviceLease{},
		sessions:        map[string]*androidSession{},
		servers:         map[string]*mobile.AppiumServer{},
		captures:        map[string]*mobile.LoggedProcess{},
		fs:              afs.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action:           "doctor",
		RequestInfo:      &endly.ActionInfo{Description: "check Android emulator runner prerequisites"},
		RequestProvider:  func() interface{} { return &DoctorRequest{} },
		ResponseProvider: func() interface{} { return &DoctorResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.doctor(ctx, request.(*DoctorRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "device-start",
		RequestInfo:      &endly.ActionInfo{Description: "attach to a device or start and lease an Android emulator"},
		RequestProvider:  func() interface{} { return &DeviceStartRequest{} },
		ResponseProvider: func() interface{} { return &DeviceStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceStart(ctx, request.(*DeviceStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "device-stop",
		RequestInfo:      &endly.ActionInfo{Description: "release an Android device lease and stop an owned emulator"},
		RequestProvider:  func() interface{} { return &DeviceStopRequest{} },
		ResponseProvider: func() interface{} { return &DeviceStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceStop(ctx, request.(*DeviceStopRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "open",
		RequestInfo:      &endly.ActionInfo{Description: "open an Android UiAutomator2 session"},
		RequestProvider:  func() interface{} { return &OpenRequest{} },
		ResponseProvider: func() interface{} { return &OpenResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.open(ctx, request.(*OpenRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "run",
		RequestInfo:      &endly.ActionInfo{Description: "run Android application DSL commands"},
		RequestProvider:  func() interface{} { return &RunRequest{} },
		ResponseProvider: func() interface{} { return &RunResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.run(ctx, request.(*RunRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "close",
		RequestInfo:      &endly.ActionInfo{Description: "close an Android UiAutomator2 session"},
		RequestProvider:  func() interface{} { return &CloseRequest{} },
		ResponseProvider: func() interface{} { return &CloseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.close(ctx.Background(), request.(*CloseRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "server-start",
		RequestInfo:      &endly.ActionInfo{Description: "start or register an Appium server for Android"},
		RequestProvider:  func() interface{} { return &ServerStartRequest{} },
		ResponseProvider: func() interface{} { return &ServerStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.serverStart(ctx, request.(*ServerStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "server-stop",
		RequestInfo:      &endly.ActionInfo{Description: "stop an owned Android Appium server"},
		RequestProvider:  func() interface{} { return &ServerStopRequest{} },
		ResponseProvider: func() interface{} { return &ServerStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.serverStop(ctx.Background(), request.(*ServerStopRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "install",
		RequestInfo:      &endly.ActionInfo{Description: "install an APK on a fenced Android device"},
		RequestProvider:  func() interface{} { return &InstallRequest{} },
		ResponseProvider: func() interface{} { return &InstallResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.install(ctx, request.(*InstallRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "uninstall",
		RequestInfo:      &endly.ActionInfo{Description: "uninstall an exact package from a fenced Android device"},
		RequestProvider:  func() interface{} { return &UninstallRequest{} },
		ResponseProvider: func() interface{} { return &UninstallResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.uninstall(ctx, request.(*UninstallRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "launch",
		RequestInfo:      &endly.ActionInfo{Description: "resolve and launch an Android application"},
		RequestProvider:  func() interface{} { return &LaunchRequest{} },
		ResponseProvider: func() interface{} { return &LaunchResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.launch(ctx, request.(*LaunchRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "terminate",
		RequestInfo:      &endly.ActionInfo{Description: "force-stop an exact Android package"},
		RequestProvider:  func() interface{} { return &TerminateRequest{} },
		ResponseProvider: func() interface{} { return &TerminateResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.terminate(ctx, request.(*TerminateRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "artifact",
		RequestInfo:      &endly.ActionInfo{Description: "capture Android screenshot and UI hierarchy evidence"},
		RequestProvider:  func() interface{} { return &ArtifactRequest{} },
		ResponseProvider: func() interface{} { return &ArtifactResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.artifact(ctx, request.(*ArtifactRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "build",
		RequestInfo:      &endly.ActionInfo{Description: "build Android project artifacts with its Gradle wrapper"},
		RequestProvider:  func() interface{} { return &BuildRequest{} },
		ResponseProvider: func() interface{} { return &BuildResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.build(ctx, request.(*BuildRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "cleanup",
		RequestInfo:      &endly.ActionInfo{Description: "clean all Android resources in LIFO order"},
		RequestProvider:  func() interface{} { return &CleanupRequest{} },
		ResponseProvider: func() interface{} { return &CleanupResponse{} },
		Handler: func(ctx *endly.Context, _ interface{}) (interface{}, error) {
			return &CleanupResponse{Errors: s.cleanupStack(ctx).Close(context.Background())}, nil
		},
	})
	s.Register(&endly.Route{
		Action:           "test",
		RequestInfo:      &endly.ActionInfo{Description: "install and run Android instrumentation tests"},
		RequestProvider:  func() interface{} { return &TestRequest{} },
		ResponseProvider: func() interface{} { return &TestResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.test(ctx, request.(*TestRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "capture-start",
		RequestInfo:      &endly.ActionInfo{Description: "start owned Android logcat capture"},
		RequestProvider:  func() interface{} { return &CaptureStartRequest{} },
		ResponseProvider: func() interface{} { return &CaptureStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.captureStart(ctx, request.(*CaptureStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "capture-stop",
		RequestInfo:      &endly.ActionInfo{Description: "stop Android logcat capture"},
		RequestProvider:  func() interface{} { return &CaptureStopRequest{} },
		ResponseProvider: func() interface{} { return &CaptureStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.captureStop(ctx.Background(), request.(*CaptureStopRequest))
		},
	})
}

func (s *service) launch(ctx *endly.Context, request *LaunchRequest) (*LaunchResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	component := request.Activity
	if component == "" {
		resolved, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "cmd", "package", "resolve-activity", "--brief", request.Package}})
		if err != nil || strings.TrimSpace(resolved.Stdout) == "" {
			return nil, fmt.Errorf("resolve launch activity for %s", request.Package)
		}
		component = strings.TrimSpace(resolved.Stdout)
	} else if !strings.Contains(component, "/") {
		component = request.Package + "/" + component
	}
	result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "am", "start", "-W", "-n", component}})
	if err != nil {
		return nil, fmt.Errorf("launch Android component %s: %w", component, err)
	}
	return &LaunchResponse{Component: component, Output: result.Stdout + result.Stderr}, nil
}

func (s *service) terminate(ctx *endly.Context, request *TerminateRequest) (*TerminateResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "am", "force-stop", request.Package}}); err != nil {
		return nil, fmt.Errorf("terminate Android package %s: %w", request.Package, err)
	}
	return &TerminateResponse{Terminated: true}, nil
}

func (s *service) captureStart(ctx *endly.Context, request *CaptureStartRequest) (*CaptureStartResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if request.Clear {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "logcat", "-c"}}); err != nil {
			return nil, fmt.Errorf("clear logcat: %w", err)
		}
	}
	args := []string{"-s", request.Lease.Serial, "logcat", "-v", "threadtime"}
	if request.Package != "" {
		pid, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "pidof", "-s", request.Package}})
		if err != nil || strings.TrimSpace(pid.Stdout) == "" {
			return nil, fmt.Errorf("resolve Android package %s process ID", request.Package)
		}
		args = append(args, "--pid", strings.TrimSpace(pid.Stdout))
	}
	process, err := mobile.StartLoggedProcess(ctx.Background(), s.runner, mobile.Command{Name: adb, Args: args}, request.LogPath)
	if err != nil {
		return nil, err
	}
	handle := CaptureHandle{ID: "android-capture-" + uuid.NewString(), Lease: request.Lease, PID: process.PID, LogPath: request.LogPath}
	s.mu.Lock()
	s.captures[handle.ID] = process
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("capture:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.captureStop(cleanupCtx, &CaptureStopRequest{Capture: handle})
		return err
	})
	return &CaptureStartResponse{Capture: handle}, nil
}

func (s *service) captureStop(ctx context.Context, request *CaptureStopRequest) (*CaptureStopResponse, error) {
	s.mu.Lock()
	process, ok := s.captures[request.Capture.ID]
	if ok {
		delete(s.captures, request.Capture.ID)
	}
	s.mu.Unlock()
	if !ok {
		return &CaptureStopResponse{Warning: "capture already stopped or unknown"}, nil
	}
	if err := s.validateLease(request.Capture.Lease); err != nil {
		return nil, err
	}
	if err := process.Stop(ctx); err != nil && err != context.DeadlineExceeded {
		return nil, err
	}
	info, _ := os.Stat(process.Path)
	size := 0
	if info != nil {
		size = int(info.Size())
	}
	return &CaptureStopResponse{Stopped: true, Artifact: &mobile.Evidence{Kind: "logcat", URL: process.Path, Size: size, Sensitive: true}}, nil
}

func (s *service) artifact(ctx *endly.Context, request *ArtifactRequest) (*ArtifactResponse, error) {
	if request.SessionID == "" {
		if request.Lease == nil {
			return nil, fmt.Errorf("Lease is required")
		}
		if err := s.validateLease(*request.Lease); err != nil {
			return nil, err
		}
		adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
		if err != nil {
			return nil, err
		}
		captured, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "exec-out", "screencap", "-p"}})
		if err != nil {
			return nil, fmt.Errorf("capture Android emulator screenshot: %w", err)
		}
		name := "android-" + safeArtifactPart(request.Lease.Serial) + "-" + uuid.NewString() + ".png"
		evidence, err := mobile.WriteEvidence(ctx.Background(), s.fs, ctx.Expand(request.Directory), name, "screenshot", []byte(captured.Stdout), true)
		if err != nil {
			return nil, err
		}
		return &ArtifactResponse{Artifacts: []*mobile.Evidence{evidence}}, nil
	}
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("Android session %q was not found", request.SessionID)
	}
	if err := s.validateLease(session.handle.Lease); err != nil {
		return nil, err
	}
	result := &ArtifactResponse{Artifacts: []*mobile.Evidence{}}
	prefix := "android-" + safeArtifactPart(request.SessionID) + "-" + uuid.NewString()
	session.mu.Lock()
	defer session.mu.Unlock()
	if request.Screenshot {
		data, err := session.appium.Screenshot(ctx.Background())
		if err != nil {
			return result, err
		}
		evidence, err := mobile.WriteEvidence(ctx.Background(), s.fs, ctx.Expand(request.Directory), prefix+".png", "screenshot", data, true)
		if err != nil {
			return result, err
		}
		result.Artifacts = append(result.Artifacts, evidence)
	}
	if request.PageSource {
		source, err := session.appium.PageSource(ctx.Background())
		if err != nil {
			return result, err
		}
		if len(source) > request.MaxSourceBytes {
			source = source[:request.MaxSourceBytes] + "\n<!-- truncated by Endly -->"
		}
		evidence, err := mobile.WriteEvidence(ctx.Background(), s.fs, ctx.Expand(request.Directory), prefix+".xml", "pageSource", []byte(source), true)
		if err != nil {
			return result, err
		}
		result.Artifacts = append(result.Artifacts, evidence)
	}
	return result, nil
}

func safeArtifactPart(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
}

func (s *service) install(ctx *endly.Context, request *InstallRequest) (*InstallResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if request.State == "freshInstall" {
		result, uninstallErr := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "uninstall", request.Package}})
		if uninstallErr != nil {
			detail := strings.ToLower(result.Stdout + result.Stderr + uninstallErr.Error())
			if !strings.Contains(detail, "unknown package") && !strings.Contains(detail, "not installed") {
				return nil, fmt.Errorf("remove existing Android package %s: %w", request.Package, uninstallErr)
			}
		}
	}
	args := []string{"-s", request.Lease.Serial, "install"}
	if request.State != "freshInstall" {
		args = append(args, "-r")
	}
	if request.GrantAll {
		args = append(args, "-g")
	}
	if request.AllowTest {
		args = append(args, "-t")
	}
	if request.AllowDowngrade {
		args = append(args, "-d")
	}
	args = append(args, request.APKPath)
	if result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: args}); err != nil {
		return nil, fmt.Errorf("install Android package %s: %s: %w", request.Package, strings.TrimSpace(result.Stderr), err)
	}
	if request.State == "cleanData" {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "pm", "clear", request.Package}}); err != nil {
			return nil, fmt.Errorf("clear Android package %s: %w", request.Package, err)
		}
	}
	return &InstallResponse{Installed: true, Package: request.Package, APKPath: request.APKPath, State: request.State}, nil
}

func (s *service) uninstall(ctx *endly.Context, request *UninstallRequest) (*UninstallResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "uninstall", request.Package}}); err != nil {
		return nil, fmt.Errorf("uninstall Android package %s: %s: %w", request.Package, strings.TrimSpace(result.Stderr), err)
	}
	return &UninstallResponse{Removed: true, Package: request.Package}, nil
}

func (s *service) serverStart(ctx *endly.Context, request *ServerStartRequest) (*ServerStartResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	server, err := mobile.StartAppium(ctx.Background(), s.runner, request.options())
	if err != nil {
		return nil, err
	}
	server.ID = "android-server-" + uuid.NewString()
	handle := ServerHandle{ID: server.ID, Endpoint: server.Endpoint, Ownership: server.Ownership, PID: server.PID, LogPath: server.LogPath}
	s.mu.Lock()
	s.servers[handle.ID] = server
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("server:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.serverStop(cleanupCtx, &ServerStopRequest{Server: handle})
		return err
	})
	return &ServerStartResponse{Server: handle}, nil
}

func (s *service) serverStop(ctx context.Context, request *ServerStopRequest) (*ServerStopResponse, error) {
	s.mu.Lock()
	server, ok := s.servers[request.Server.ID]
	if ok {
		if server.Endpoint != request.Server.Endpoint || server.Ownership != request.Server.Ownership {
			s.mu.Unlock()
			return nil, fmt.Errorf("Appium server handle mismatch")
		}
		delete(s.servers, request.Server.ID)
	}
	s.mu.Unlock()
	if !ok {
		return &ServerStopResponse{Warning: "server already stopped or unknown"}, nil
	}
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := server.Stop(stopCtx); err != nil && err != context.DeadlineExceeded {
		return nil, err
	}
	return &ServerStopResponse{Stopped: server.Ownership == "managed"}, nil
}

func (s *service) open(ctx *endly.Context, request *OpenRequest) (*OpenResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	if err := s.validateServer(request.Server); err != nil {
		return nil, err
	}
	capabilities := map[string]interface{}{}
	for key, value := range request.Capabilities {
		if isProtectedAndroidCapability(key) {
			return nil, fmt.Errorf("capability %q is owned by the Android service", key)
		}
		capabilities[key] = value
	}
	capabilities["platformName"] = "Android"
	capabilities["appium:automationName"] = "UiAutomator2"
	capabilities["appium:udid"] = request.Lease.Serial
	capabilities["appium:appPackage"] = request.Package
	if request.Activity != "" {
		capabilities["appium:appActivity"] = request.Activity
	}
	client, err := mobile.NewAppiumClient(ctx.Expand(request.Server.Endpoint), nil)
	if err != nil {
		return nil, err
	}
	if _, err := client.Status(ctx.Background()); err != nil {
		return nil, fmt.Errorf("Appium health check: %w", err)
	}
	appiumSession, err := client.NewSession(ctx.Background(), capabilities)
	if err != nil {
		return nil, err
	}
	handle := SessionHandle{ID: request.SessionID, BackendSessionID: appiumSession.ID, Lease: request.Lease, Server: request.Server}
	owned := &androidSession{handle: handle, appium: appiumSession, testIDStrategy: request.TestIDStrategy}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		_ = appiumSession.Close(context.Background())
		return nil, fmt.Errorf("Android session %q already exists", handle.ID)
	}
	s.sessions[handle.ID] = owned
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("session:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.close(cleanupCtx, &CloseRequest{SessionID: handle.ID})
		return err
	})
	return &OpenResponse{Session: handle}, nil
}

func (s *service) run(ctx *endly.Context, request *RunRequest) (*RunResponse, error) {
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("Android session %q was not found", request.SessionID)
	}
	if err := s.validateLease(session.handle.Lease); err != nil {
		return nil, err
	}
	commands := make([]string, len(request.Commands))
	for i, command := range request.Commands {
		commands[i] = ctx.Expand(command)
	}
	executor := &mobile.Executor{
		Session:       session.appium,
		Resolve:       androidLocatorResolver(session.testIDStrategy),
		ExecuteDevice: executeAndroidDevice,
		ActionTimeout: time.Duration(request.ActionTimeoutMs) * time.Millisecond,
		PollInterval:  time.Duration(request.PollIntervalMs) * time.Millisecond,
	}
	session.mu.Lock()
	result, err := executor.Run(ctx.Background(), commands)
	session.mu.Unlock()
	if result == nil {
		return nil, err
	}
	return &RunResponse{Data: result.Data, Steps: result.Steps, Validations: result.Validations}, err
}

func (s *service) close(ctx context.Context, request *CloseRequest) (*CloseResponse, error) {
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	if ok {
		delete(s.sessions, request.SessionID)
	}
	s.mu.Unlock()
	if !ok {
		return &CloseResponse{Warning: "session already closed or unknown"}, nil
	}
	session.mu.Lock()
	err := session.appium.Close(ctx)
	session.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &CloseResponse{Closed: true}, nil
}

func (s *service) validateLease(lease DeviceLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.leases[lease.ID]
	if !ok {
		return fmt.Errorf("Android device lease %q was not found", lease.ID)
	}
	if stored.Fence != lease.Fence || stored.Serial != lease.Serial {
		return fmt.Errorf("Android device lease fence mismatch")
	}
	return nil
}

func (s *service) validateServer(handle ServerHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	server, ok := s.servers[handle.ID]
	if !ok {
		return fmt.Errorf("Android Appium server %q was not found", handle.ID)
	}
	if server.Endpoint != handle.Endpoint || server.Ownership != handle.Ownership {
		return fmt.Errorf("Android Appium server handle mismatch")
	}
	return nil
}

func androidLocatorResolver(testIDStrategy string) mobile.LocatorResolver {
	return func(call mobile.Call) (mobile.Locator, bool, error) {
		value, err := stringCallArg(call, 0)
		if err != nil {
			return mobile.Locator{}, true, err
		}
		switch strings.ToLower(call.Name) {
		case "getbytestid":
			if testIDStrategy == "accessibilityId" {
				return mobile.Locator{Using: "accessibility id", Value: value}, true, nil
			}
			return mobile.Locator{Using: "id", Value: value}, true, nil
		case "getbyaccessibilityid":
			return mobile.Locator{Using: "accessibility id", Value: value}, true, nil
		case "getbyresourceid":
			return mobile.Locator{Using: "id", Value: value}, true, nil
		case "getbytext":
			return mobile.Locator{Using: "-android uiautomator", Value: "new UiSelector().text(" + strconv.Quote(value) + ")"}, true, nil
		case "getbyclass":
			return mobile.Locator{Using: "class name", Value: value}, true, nil
		case "getbyxpath":
			return mobile.Locator{Using: "xpath", Value: value}, true, nil
		default:
			return mobile.Locator{}, false, nil
		}
	}
}

func executeAndroidDevice(ctx context.Context, session *mobile.AppiumSession, call mobile.Call) (interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "back":
		return session.Execute(ctx, "mobile: pressKey", map[string]interface{}{"keycode": 4})
	case "home":
		return session.Execute(ctx, "mobile: pressKey", map[string]interface{}{"keycode": 3})
	case "hidekeyboard":
		return session.Execute(ctx, "mobile: hideKeyboard")
	default:
		return nil, fmt.Errorf("unsupported Android device command %q", call.Name)
	}
}

func stringCallArg(call mobile.Call, index int) (string, error) {
	if index >= len(call.Args) {
		return "", fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	value, ok := call.Args[index].(string)
	if !ok {
		return "", fmt.Errorf("%s argument %d must be a string", call.Name, index+1)
	}
	return value, nil
}

func isProtectedAndroidCapability(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "platformname", "appium:automationname", "appium:udid", "appium:apppackage", "appium:appactivity", "udid", "automationname", "apppackage", "appactivity":
		return true
	default:
		return false
	}
}

func (s *service) doctor(ctx *endly.Context, request *DoctorRequest) (*DoctorResponse, error) {
	result := &DoctorResponse{AndroidSDKRoot: request.AndroidSDKRoot}
	required := map[string]bool{}
	for _, name := range request.Required {
		required[strings.ToLower(strings.TrimSpace(name))] = true
	}

	adb, adbErr := resolveAndroidTool("adb", request.AndroidSDKRoot)
	result.Checks = append(result.Checks, s.toolCheck(ctx.Background(), "adb", adb, adbErr, []string{"version"}, "install Android SDK platform-tools and set ANDROID_SDK_ROOT"))
	if adbErr == nil {
		if run, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"devices", "-l"}}); err == nil {
			result.Devices = parseADBDevices(run.Stdout)
		}
	}

	emulator, emulatorErr := resolveAndroidTool("emulator", request.AndroidSDKRoot)
	result.Checks = append(result.Checks, s.toolCheck(ctx.Background(), "emulator", emulator, emulatorErr, []string{"-version"}, "install the Android emulator package"))
	if emulatorErr == nil {
		if run, err := s.runner.Run(ctx.Background(), mobile.Command{Name: emulator, Args: []string{"-list-avds"}}); err == nil {
			for _, line := range strings.Split(run.Stdout, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					result.AVDs = append(result.AVDs, line)
				}
			}
			sort.Strings(result.AVDs)
		}
	}

	appium, appiumErr := mobile.ResolveExecutable("appium")
	result.Checks = append(result.Checks, s.toolCheck(ctx.Background(), "appium", appium, appiumErr, []string{"--version"}, "install the pinned Appium distribution"))
	result.Ready = mobile.ChecksReady(result.Checks, required)
	return result, nil
}

func (s *service) toolCheck(ctx context.Context, name, path string, resolveErr error, args []string, remediation string) mobile.Check {
	check := mobile.Check{Name: name, Path: path, Status: "missing", Remediation: remediation}
	if resolveErr != nil {
		check.Detail = resolveErr.Error()
		return check
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := s.runner.Run(probeCtx, mobile.Command{Name: path, Args: args})
	if err != nil {
		check.Status = "error"
		check.Detail = strings.TrimSpace(result.Stderr + " " + err.Error())
		return check
	}
	check.Status = "ok"
	check.Version = firstLine(result.Stdout + result.Stderr)
	return check
}

func (s *service) deviceStart(ctx *endly.Context, request *DeviceStartRequest) (*DeviceStartResponse, error) {
	adb, err := resolveAndroidTool("adb", request.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if request.Serial != "" {
		devices, err := s.adbDevices(ctx.Background(), adb)
		if err != nil {
			return nil, err
		}
		for _, device := range devices {
			if device.Serial == request.Serial && device.State == "device" {
				lease := DeviceLease{ID: uuid.NewString(), Fence: 1, Serial: request.Serial, AndroidSDKRoot: request.AndroidSDKRoot}
				s.storeLease(lease)
				return &DeviceStartResponse{Lease: lease}, nil
			}
		}
		return nil, fmt.Errorf("Android device %q is not connected and ready", request.Serial)
	}

	emulator, err := resolveAndroidTool("emulator", request.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	port, err := s.selectPort(ctx.Background(), adb, request.Port)
	if err != nil {
		return nil, err
	}
	serial := "emulator-" + strconv.Itoa(port)
	args := []string{"-avd", request.AVD, "-port", strconv.Itoa(port)}
	if request.WipeData {
		args = append(args, "-wipe-data")
	}
	if request.NoWindow {
		args = append(args, "-no-window")
	}
	if request.NoSnapshot {
		args = append(args, "-no-snapshot")
	}
	args = append(args, request.EmulatorArgs...)

	stdout, stderr, closeLog, err := emulatorLog(request.LogPath)
	if err != nil {
		return nil, err
	}
	process, err := s.runner.Start(ctx.Background(), mobile.Command{Name: emulator, Args: args}, stdout, stderr)
	if err != nil {
		closeLog()
		return nil, err
	}
	go func() {
		<-process.Done
		closeLog()
	}()

	if err := s.waitForBoot(ctx.Background(), adb, serial, request.BootTimeoutMs, request.PollIntervalMs); err != nil {
		_, _ = s.runner.Run(context.Background(), mobile.Command{Name: adb, Args: []string{"-s", serial, "emu", "kill"}})
		return nil, err
	}
	if request.Animations != nil && !*request.Animations {
		if err := s.setAnimations(ctx.Background(), adb, serial, false); err != nil {
			_, _ = s.runner.Run(context.Background(), mobile.Command{Name: adb, Args: []string{"-s", serial, "emu", "kill"}})
			return nil, err
		}
	}
	lease := DeviceLease{ID: uuid.NewString(), Fence: 1, Serial: serial, AVD: request.AVD, PID: process.PID, Owned: true, LogPath: request.LogPath, AndroidSDKRoot: request.AndroidSDKRoot}
	s.storeLease(lease)
	s.cleanupStack(ctx).Push("device:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.stopOwned(cleanupCtx, adb, lease)
		return err
	})
	return &DeviceStartResponse{Lease: lease}, nil
}

func (s *service) cleanupStack(ctx *endly.Context) *mobile.CleanupStack {
	holder := &contextCleanup{}
	if ctx.GetInto(cleanupKey, &holder) && holder.stack != nil {
		return holder.stack
	}
	holder = &contextCleanup{stack: &mobile.CleanupStack{}}
	ctx.Put(cleanupKey, holder)
	ctx.Deffer(func() { holder.stack.Close(context.Background()) })
	return holder.stack
}

func (s *service) deviceStop(ctx *endly.Context, request *DeviceStopRequest) (*DeviceStopResponse, error) {
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	return s.stopOwned(ctx.Background(), adb, request.Lease)
}

func (s *service) stopOwned(ctx context.Context, adb string, lease DeviceLease) (*DeviceStopResponse, error) {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	if !ok {
		s.mu.Unlock()
		return &DeviceStopResponse{Warning: "lease already released or unknown"}, nil
	}
	if stored.Fence != lease.Fence || stored.Serial != lease.Serial {
		s.mu.Unlock()
		return nil, fmt.Errorf("device lease fence mismatch")
	}
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	if !stored.Owned {
		return &DeviceStopResponse{Warning: "attached physical/external device was released but not stopped"}, nil
	}
	_, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: []string{"-s", stored.Serial, "emu", "kill"}})
	if err != nil {
		return nil, fmt.Errorf("stop emulator %s: %w", stored.Serial, err)
	}
	return &DeviceStopResponse{Stopped: true}, nil
}

func (s *service) storeLease(lease DeviceLease) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leases[lease.ID] = lease
}

func (s *service) adbDevices(ctx context.Context, adb string) ([]AndroidDevice, error) {
	result, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: []string{"devices", "-l"}})
	if err != nil {
		return nil, err
	}
	return parseADBDevices(result.Stdout), nil
}

func (s *service) selectPort(ctx context.Context, adb string, requested int) (int, error) {
	devices, err := s.adbDevices(ctx, adb)
	if err != nil {
		return 0, err
	}
	used := map[int]bool{}
	for _, device := range devices {
		if strings.HasPrefix(device.Serial, "emulator-") {
			port, _ := strconv.Atoi(strings.TrimPrefix(device.Serial, "emulator-"))
			used[port] = true
		}
	}
	if requested != 0 {
		if used[requested] {
			return 0, fmt.Errorf("emulator port %d is already in use", requested)
		}
		return requested, nil
	}
	for port := 5554; port <= 5682; port += 2 {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free Android emulator console port")
}

func (s *service) waitForBoot(parent context.Context, adb, serial string, timeoutMs, pollMs int) error {
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	ticker := time.NewTicker(time.Duration(pollMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		probe, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: []string{"-s", serial, "shell", "getprop", "sys.boot_completed"}})
		if err == nil && strings.TrimSpace(probe.Stdout) == "1" {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Android emulator %s boot: %w", serial, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *service) setAnimations(ctx context.Context, adb, serial string, enabled bool) error {
	value := "0"
	if enabled {
		value = "1"
	}
	for _, setting := range []string{"window_animation_scale", "transition_animation_scale", "animator_duration_scale"} {
		if _, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: []string{"-s", serial, "shell", "settings", "put", "global", setting, value}}); err != nil {
			return fmt.Errorf("set %s: %w", setting, err)
		}
	}
	return nil
}

func resolveAndroidTool(name, root string) (string, error) {
	candidates := []string{}
	if root != "" {
		switch name {
		case "adb":
			candidates = append(candidates, filepath.Join(root, "platform-tools", "adb"))
		case "emulator":
			candidates = append(candidates, filepath.Join(root, "emulator", "emulator"))
		}
	}
	return mobile.ResolveExecutable(name, candidates...)
}

func parseADBDevices(output string) []AndroidDevice {
	result := []AndroidDevice{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		result = append(result, AndroidDevice{Serial: fields[0], State: fields[1], Detail: strings.Join(fields[2:], " ")})
	}
	return result
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}

func emulatorLog(path string) (io.Writer, io.Writer, func(), error) {
	if path == "" {
		return io.Discard, io.Discard, func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, nil, fmt.Errorf("create emulator log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open emulator log: %w", err)
	}
	return file, file, func() { _ = file.Close() }, nil
}
