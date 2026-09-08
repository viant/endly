package ios

import (
	"context"
	"encoding/json"
	"fmt"
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

const ServiceID = "ios"

type service struct {
	*endly.AbstractService
	runner   mobile.Runner
	mu       sync.Mutex
	leases   map[string]DestinationLease
	sessions map[string]*iosSession
	servers  map[string]*iosServer
	captures map[string]*mobile.LoggedProcess
	fs       afs.Service
}

type iosSession struct {
	handle SessionHandle
	appium *mobile.AppiumSession
	mu     sync.Mutex
}

type iosServer struct {
	appium      *mobile.AppiumServer
	destination DestinationLease
	appiumHome  string
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
		leases:          map[string]DestinationLease{},
		sessions:        map[string]*iosSession{},
		servers:         map[string]*iosServer{},
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
		RequestInfo:      &endly.ActionInfo{Description: "check iOS Simulator runner prerequisites"},
		RequestProvider:  func() interface{} { return &DoctorRequest{} },
		ResponseProvider: func() interface{} { return &DoctorResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.doctor(ctx, request.(*DoctorRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "simulator-start",
		RequestInfo:      &endly.ActionInfo{Description: "attach to, clone, or create and boot an iOS Simulator"},
		RequestProvider:  func() interface{} { return &SimulatorStartRequest{} },
		ResponseProvider: func() interface{} { return &SimulatorStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.simulatorStart(ctx, request.(*SimulatorStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "simulator-stop",
		RequestInfo:      &endly.ActionInfo{Description: "release an iOS Simulator lease and delete an owned clone"},
		RequestProvider:  func() interface{} { return &SimulatorStopRequest{} },
		ResponseProvider: func() interface{} { return &SimulatorStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.simulatorStop(ctx, request.(*SimulatorStopRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "open",
		RequestInfo:      &endly.ActionInfo{Description: "open an iOS XCUITest session"},
		RequestProvider:  func() interface{} { return &OpenRequest{} },
		ResponseProvider: func() interface{} { return &OpenResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.open(ctx, request.(*OpenRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "run",
		RequestInfo:      &endly.ActionInfo{Description: "run iOS application DSL commands"},
		RequestProvider:  func() interface{} { return &RunRequest{} },
		ResponseProvider: func() interface{} { return &RunResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.run(ctx, request.(*RunRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "close",
		RequestInfo:      &endly.ActionInfo{Description: "close an iOS XCUITest session"},
		RequestProvider:  func() interface{} { return &CloseRequest{} },
		ResponseProvider: func() interface{} { return &CloseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.close(ctx.Background(), request.(*CloseRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "server-start",
		RequestInfo:      &endly.ActionInfo{Description: "start or register an Appium server for iOS"},
		RequestProvider:  func() interface{} { return &ServerStartRequest{} },
		ResponseProvider: func() interface{} { return &ServerStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.serverStart(ctx, request.(*ServerStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "server-stop",
		RequestInfo:      &endly.ActionInfo{Description: "stop an owned iOS Appium server"},
		RequestProvider:  func() interface{} { return &ServerStopRequest{} },
		ResponseProvider: func() interface{} { return &ServerStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.serverStop(ctx.Background(), request.(*ServerStopRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "install",
		RequestInfo:      &endly.ActionInfo{Description: "install an app on a fenced iOS Simulator"},
		RequestProvider:  func() interface{} { return &InstallRequest{} },
		ResponseProvider: func() interface{} { return &InstallResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.install(ctx, request.(*InstallRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "uninstall",
		RequestInfo:      &endly.ActionInfo{Description: "uninstall an exact bundle from a fenced iOS Simulator"},
		RequestProvider:  func() interface{} { return &UninstallRequest{} },
		ResponseProvider: func() interface{} { return &UninstallResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.uninstall(ctx, request.(*UninstallRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "launch",
		RequestInfo:      &endly.ActionInfo{Description: "launch an exact bundle on an iOS Simulator"},
		RequestProvider:  func() interface{} { return &LaunchRequest{} },
		ResponseProvider: func() interface{} { return &LaunchResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.launch(ctx, request.(*LaunchRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "terminate",
		RequestInfo:      &endly.ActionInfo{Description: "terminate an exact bundle on an iOS Simulator"},
		RequestProvider:  func() interface{} { return &TerminateRequest{} },
		ResponseProvider: func() interface{} { return &TerminateResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.terminate(ctx, request.(*TerminateRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "artifact",
		RequestInfo:      &endly.ActionInfo{Description: "capture iOS screenshot and accessibility hierarchy evidence"},
		RequestProvider:  func() interface{} { return &ArtifactRequest{} },
		ResponseProvider: func() interface{} { return &ArtifactResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.artifact(ctx, request.(*ArtifactRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "build",
		RequestInfo:      &endly.ActionInfo{Description: "build iOS Simulator products with xcodebuild"},
		RequestProvider:  func() interface{} { return &BuildRequest{} },
		ResponseProvider: func() interface{} { return &BuildResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.build(ctx, request.(*BuildRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "cleanup",
		RequestInfo:      &endly.ActionInfo{Description: "clean all iOS resources in LIFO order"},
		RequestProvider:  func() interface{} { return &CleanupRequest{} },
		ResponseProvider: func() interface{} { return &CleanupResponse{} },
		Handler: func(ctx *endly.Context, _ interface{}) (interface{}, error) {
			return &CleanupResponse{Errors: s.cleanupStack(ctx).Close(context.Background())}, nil
		},
	})
	s.Register(&endly.Route{
		Action:           "test",
		RequestInfo:      &endly.ActionInfo{Description: "run XCTest and normalize xcresult summary"},
		RequestProvider:  func() interface{} { return &TestRequest{} },
		ResponseProvider: func() interface{} { return &TestResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.test(ctx, request.(*TestRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "capture-start",
		RequestInfo:      &endly.ActionInfo{Description: "start owned iOS Simulator unified-log capture"},
		RequestProvider:  func() interface{} { return &CaptureStartRequest{} },
		ResponseProvider: func() interface{} { return &CaptureStartResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.captureStart(ctx, request.(*CaptureStartRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "capture-stop",
		RequestInfo:      &endly.ActionInfo{Description: "stop iOS Simulator unified-log capture"},
		RequestProvider:  func() interface{} { return &CaptureStopRequest{} },
		ResponseProvider: func() interface{} { return &CaptureStopResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.captureStop(ctx.Background(), request.(*CaptureStopRequest))
		},
	})
}

func (s *service) launch(ctx *endly.Context, request *LaunchRequest) (*LaunchResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	args := []string{"simctl", "launch", "--terminate-running-process", request.Destination.UDID, request.BundleID}
	args = append(args, request.Arguments...)
	env := map[string]string{}
	for key, value := range request.Environment {
		env["SIMCTL_CHILD_"+key] = value
	}
	result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: args, Env: env})
	if err != nil {
		return nil, fmt.Errorf("launch iOS bundle %s: %w", request.BundleID, err)
	}
	output := strings.TrimSpace(result.Stdout + result.Stderr)
	pid := 0
	if index := strings.LastIndex(output, ":"); index >= 0 {
		pid, _ = strconv.Atoi(strings.TrimSpace(output[index+1:]))
	}
	return &LaunchResponse{PID: pid, Output: output}, nil
}

func (s *service) terminate(ctx *endly.Context, request *TerminateRequest) (*TerminateResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "terminate", request.Destination.UDID, request.BundleID}}); err != nil {
		return nil, fmt.Errorf("terminate iOS bundle %s: %w", request.BundleID, err)
	}
	return &TerminateResponse{Terminated: true}, nil
}

func (s *service) captureStart(ctx *endly.Context, request *CaptureStartRequest) (*CaptureStartResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	args := []string{"simctl", "spawn", request.Destination.UDID, "log", "stream", "--style", "ndjson", "--level", "info"}
	if request.Predicate != "" {
		args = append(args, "--predicate", request.Predicate)
	}
	process, err := mobile.StartLoggedProcess(ctx.Background(), s.runner, mobile.Command{Name: xcrun, Args: args}, request.LogPath)
	if err != nil {
		return nil, err
	}
	handle := CaptureHandle{ID: "ios-capture-" + uuid.NewString(), Destination: request.Destination, PID: process.PID, LogPath: request.LogPath}
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
	if err := s.validateLease(request.Capture.Destination); err != nil {
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
	return &CaptureStopResponse{Stopped: true, Artifact: &mobile.Evidence{Kind: "unifiedLog", URL: process.Path, Size: size, Sensitive: true}}, nil
}

func (s *service) artifact(ctx *endly.Context, request *ArtifactRequest) (*ArtifactResponse, error) {
	if request.SessionID == "" {
		if request.Destination == nil {
			return nil, fmt.Errorf("Destination is required")
		}
		if err := s.validateLease(*request.Destination); err != nil {
			return nil, err
		}
		xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
		if err != nil {
			return nil, err
		}
		tempPath := filepath.Join(os.TempDir(), "endly-ios-"+uuid.NewString()+".png")
		defer os.Remove(tempPath)
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "io", request.Destination.UDID, "screenshot", tempPath}}); err != nil {
			return nil, fmt.Errorf("capture iOS Simulator screenshot: %w", err)
		}
		data, err := os.ReadFile(tempPath)
		if err != nil {
			return nil, fmt.Errorf("read iOS Simulator screenshot: %w", err)
		}
		name := "ios-" + safeArtifactPart(request.Destination.UDID) + "-" + uuid.NewString() + ".png"
		evidence, err := mobile.WriteEvidence(ctx.Background(), s.fs, ctx.Expand(request.Directory), name, "screenshot", data, true)
		if err != nil {
			return nil, err
		}
		return &ArtifactResponse{Artifacts: []*mobile.Evidence{evidence}}, nil
	}
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("iOS session %q was not found", request.SessionID)
	}
	if err := s.validateLease(session.handle.Destination); err != nil {
		return nil, err
	}
	result := &ArtifactResponse{Artifacts: []*mobile.Evidence{}}
	prefix := "ios-" + safeArtifactPart(request.SessionID) + "-" + uuid.NewString()
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
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	if request.State == "freshInstall" {
		result, uninstallErr := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "uninstall", request.Destination.UDID, request.BundleID}})
		if uninstallErr != nil {
			detail := strings.ToLower(result.Stdout + result.Stderr + uninstallErr.Error())
			if !strings.Contains(detail, "not installed") && !strings.Contains(detail, "no such file") {
				return nil, fmt.Errorf("remove existing iOS app %s: %w", request.BundleID, uninstallErr)
			}
		}
	}
	if result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "install", request.Destination.UDID, request.App.HostPath}}); err != nil {
		return nil, fmt.Errorf("install iOS app %s: %s: %w", request.BundleID, strings.TrimSpace(result.Stderr), err)
	}
	return &InstallResponse{Installed: true, BundleID: request.BundleID, AppPath: request.App.HostPath, State: request.State}, nil
}

func (s *service) uninstall(ctx *endly.Context, request *UninstallRequest) (*UninstallResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	if result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "uninstall", request.Destination.UDID, request.BundleID}}); err != nil {
		return nil, fmt.Errorf("uninstall iOS app %s: %s: %w", request.BundleID, strings.TrimSpace(result.Stderr), err)
	}
	return &UninstallResponse{Removed: true, BundleID: request.BundleID}, nil
}

func (s *service) serverStart(ctx *endly.Context, request *ServerStartRequest) (*ServerStartResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	server, err := mobile.StartAppium(ctx.Background(), s.runner, request.options())
	if err != nil {
		return nil, err
	}
	server.ID = "ios-server-" + uuid.NewString()
	handle := ServerHandle{ID: server.ID, Endpoint: server.Endpoint, Ownership: server.Ownership, PID: server.PID, LogPath: server.LogPath}
	s.mu.Lock()
	s.servers[handle.ID] = &iosServer{appium: server, destination: request.Destination, appiumHome: request.AppiumHome}
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("server:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.serverStop(cleanupCtx, &ServerStopRequest{Server: handle})
		return err
	})
	return &ServerStartResponse{Server: handle}, nil
}

func (s *service) serverStop(ctx context.Context, request *ServerStopRequest) (*ServerStopResponse, error) {
	s.mu.Lock()
	owned, ok := s.servers[request.Server.ID]
	if ok {
		if owned.appium.Endpoint != request.Server.Endpoint || owned.appium.Ownership != request.Server.Ownership {
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
	if err := owned.appium.Stop(stopCtx); err != nil && err != context.DeadlineExceeded {
		return nil, err
	}
	if owned.appium.Ownership == "managed" {
		if err := s.stopOwnedWDA(ctx, owned); err != nil {
			return nil, err
		}
	}
	return &ServerStopResponse{Stopped: owned.appium.Ownership == "managed"}, nil
}

func (s *service) open(ctx *endly.Context, request *OpenRequest) (*OpenResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	if err := s.validateServer(request.Server); err != nil {
		return nil, err
	}
	capabilities := map[string]interface{}{}
	for key, value := range request.Capabilities {
		if isProtectedIOSCapability(key) {
			return nil, fmt.Errorf("capability %q is owned by the iOS service", key)
		}
		capabilities[key] = value
	}
	capabilities["platformName"] = "iOS"
	capabilities["appium:automationName"] = "XCUITest"
	capabilities["appium:udid"] = request.Destination.UDID
	if request.BundleID != "" {
		capabilities["appium:bundleId"] = request.BundleID
	}
	if request.App != nil {
		if request.App.HostPath == "" {
			return nil, fmt.Errorf("App.HostPath is required by the Appium worker")
		}
		capabilities["appium:app"] = request.App.HostPath
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
	handle := SessionHandle{ID: request.SessionID, BackendSessionID: appiumSession.ID, Destination: request.Destination, Server: request.Server}
	owned := &iosSession{handle: handle, appium: appiumSession}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		_ = appiumSession.Close(context.Background())
		return nil, fmt.Errorf("iOS session %q already exists", handle.ID)
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
		return nil, fmt.Errorf("iOS session %q was not found", request.SessionID)
	}
	if err := s.validateLease(session.handle.Destination); err != nil {
		return nil, err
	}
	commands := make([]string, len(request.Commands))
	for i, command := range request.Commands {
		commands[i] = ctx.Expand(command)
	}
	executor := &mobile.Executor{
		Session:       session.appium,
		Resolve:       iosLocatorResolver,
		ExecuteDevice: executeIOSDevice,
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

func (s *service) validateLease(lease DestinationLease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.leases[lease.ID]
	if !ok {
		return fmt.Errorf("iOS destination lease %q was not found", lease.ID)
	}
	if stored.Fence != lease.Fence || stored.UDID != lease.UDID {
		return fmt.Errorf("iOS destination lease fence mismatch")
	}
	return nil
}

func (s *service) validateServer(handle ServerHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	server, ok := s.servers[handle.ID]
	if !ok {
		return fmt.Errorf("iOS Appium server %q was not found", handle.ID)
	}
	if server.appium.Endpoint != handle.Endpoint || server.appium.Ownership != handle.Ownership {
		return fmt.Errorf("iOS Appium server handle mismatch")
	}
	return nil
}

func (s *service) stopOwnedWDA(ctx context.Context, server *iosServer) error {
	if server == nil || server.appiumHome == "" || server.destination.UDID == "" {
		return nil
	}
	ps, err := mobile.ResolveExecutable("ps", "/bin/ps")
	if err != nil {
		return err
	}
	result, err := s.runner.Run(ctx, mobile.Command{Name: ps, Args: []string{"-axo", "pid=,command="}})
	if err != nil {
		return fmt.Errorf("inspect WDA processes: %w", err)
	}
	marker := "-destination id=" + server.destination.UDID
	home := filepath.Clean(server.appiumHome)
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "APPIUM_XCODEBUILD_WDA_MARKER=1") ||
			!strings.Contains(line, "WebDriverAgent.xcodeproj") ||
			!strings.Contains(line, marker) ||
			!strings.Contains(line, home) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, parseErr := strconv.Atoi(fields[0])
		if parseErr != nil || pid <= 1 {
			continue
		}
		if err := stopPID(ctx, s.runner, pid); err != nil {
			return fmt.Errorf("stop owned WDA xcodebuild %d: %w", pid, err)
		}
	}
	return nil
}

func stopPID(ctx context.Context, runner mobile.Runner, pid int) error {
	kill, err := mobile.ResolveExecutable("kill", "/bin/kill")
	if err != nil {
		return err
	}
	pidText := strconv.Itoa(pid)
	if _, err := runner.Run(ctx, mobile.Command{Name: kill, Args: []string{"-TERM", pidText}}); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := runner.Run(ctx, mobile.Command{Name: kill, Args: []string{"-0", pidText}}); err != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_, err = runner.Run(ctx, mobile.Command{Name: kill, Args: []string{"-KILL", pidText}})
	return err
}

func iosLocatorResolver(call mobile.Call) (mobile.Locator, bool, error) {
	value, err := stringCallArg(call, 0)
	if err != nil {
		return mobile.Locator{}, true, err
	}
	switch strings.ToLower(call.Name) {
	case "getbytestid", "getbyaccessibilityid":
		return mobile.Locator{Using: "accessibility id", Value: value}, true, nil
	case "getbyname":
		return mobile.Locator{Using: "name", Value: value}, true, nil
	case "getbylabel", "getbytext":
		return mobile.Locator{Using: "-ios predicate string", Value: "label == '" + escapePredicate(value) + "'"}, true, nil
	case "getbytype":
		return mobile.Locator{Using: "class name", Value: value}, true, nil
	case "getbypredicate":
		return mobile.Locator{Using: "-ios predicate string", Value: value}, true, nil
	case "getbyclasschain":
		return mobile.Locator{Using: "-ios class chain", Value: value}, true, nil
	case "getbyxpath":
		return mobile.Locator{Using: "xpath", Value: value}, true, nil
	default:
		return mobile.Locator{}, false, nil
	}
}

func executeIOSDevice(ctx context.Context, session *mobile.AppiumSession, call mobile.Call) (interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "home":
		return session.Execute(ctx, "mobile: pressButton", map[string]interface{}{"name": "home"})
	case "hidekeyboard":
		return session.Execute(ctx, "mobile: hideKeyboard")
	case "deeplink":
		URL, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		bundleID, err := stringCallArg(call, 1)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: deepLink", map[string]interface{}{"url": URL, "bundleId": bundleID})
	default:
		return nil, fmt.Errorf("unsupported iOS device command %q", call.Name)
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

func escapePredicate(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`)
}

func isProtectedIOSCapability(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "platformname", "appium:automationname", "appium:udid", "appium:bundleid", "appium:app", "udid", "automationname", "bundleid", "app":
		return true
	default:
		return false
	}
}

func (s *service) doctor(ctx *endly.Context, request *DoctorRequest) (*DoctorResponse, error) {
	response := &DoctorResponse{}
	required := map[string]bool{}
	for _, item := range request.Required {
		required[strings.ToLower(strings.TrimSpace(item))] = true
	}
	xcodebuild, xcodeErr := mobile.ResolveExecutable("xcodebuild", "/usr/bin/xcodebuild")
	response.Checks = append(response.Checks, s.toolCheck(ctx.Background(), "xcodebuild", xcodebuild, xcodeErr, []string{"-version"}, "install full Xcode and select it with DEVELOPER_DIR"))
	xcrun, xcrunErr := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	response.Checks = append(response.Checks, s.toolCheck(ctx.Background(), "xcrun", xcrun, xcrunErr, []string{"--version"}, "install full Xcode command-line components"))

	runtimeCheck := mobile.Check{Name: "simulator-runtime", Status: "missing", Remediation: "install at least one compatible iOS Simulator runtime in Xcode"}
	if xcrunErr == nil {
		if run, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "list", "runtimes", "available", "--json"}}); err != nil {
			runtimeCheck.Status = "error"
			runtimeCheck.Detail = err.Error()
		} else {
			response.Runtimes = parseRuntimes(run.Stdout)
			if len(response.Runtimes) > 0 {
				runtimeCheck.Status = "ok"
				runtimeCheck.Detail = fmt.Sprintf("%d available runtime(s)", len(response.Runtimes))
			}
		}
		if run, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "list", "devices", "available", "--json"}}); err == nil {
			response.Simulators, _ = parseSimulators(run.Stdout)
		}
	}
	response.Checks = append(response.Checks, runtimeCheck)
	response.Ready = mobile.ChecksReady(response.Checks, required)
	return response, nil
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

func (s *service) simulatorStart(ctx *endly.Context, request *SimulatorStartRequest) (*SimulatorStartResponse, error) {
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	list, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "list", "devices", "available", "--json"}})
	if err != nil {
		return nil, fmt.Errorf("list iOS Simulators: %w", err)
	}
	simulators, err := parseSimulators(list.Stdout)
	if err != nil {
		return nil, err
	}
	lease := DestinationLease{ID: uuid.NewString(), Fence: 1}
	state := ""
	switch {
	case request.UDID != "":
		candidate, ok := findSimulator(simulators, request.UDID, "")
		if !ok {
			return nil, fmt.Errorf("iOS Simulator %q is not available", request.UDID)
		}
		lease.UDID, lease.Name, lease.Runtime = candidate.UDID, candidate.Name, candidate.Runtime
		state = candidate.State
	case request.BaseName != "":
		base, ok := findSimulator(simulators, "", request.BaseName)
		if !ok {
			return nil, fmt.Errorf("base iOS Simulator %q is not available or is ambiguous", request.BaseName)
		}
		if request.CloneName == "" {
			lease.UDID, lease.Name, lease.Runtime = base.UDID, base.Name, base.Runtime
			state = base.State
		} else {
			clone, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "clone", base.UDID, request.CloneName}})
			if err != nil {
				return nil, fmt.Errorf("clone iOS Simulator: %w", err)
			}
			lease.UDID = strings.TrimSpace(clone.Stdout)
			lease.Name = request.CloneName
			lease.Runtime = base.Runtime
			lease.OwnedClone = true
		}
	default:
		created, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "create", request.DeviceType, request.DeviceType, request.Runtime}})
		if err != nil {
			return nil, fmt.Errorf("create iOS Simulator: %w", err)
		}
		lease.UDID = strings.TrimSpace(created.Stdout)
		lease.Name = request.DeviceType
		lease.Runtime = request.Runtime
		lease.OwnedClone = true
	}
	if lease.UDID == "" {
		return nil, fmt.Errorf("simctl returned an empty Simulator UDID")
	}
	if request.Erase {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "erase", lease.UDID}}); err != nil {
			if lease.OwnedClone {
				_, _ = s.runner.Run(context.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "delete", lease.UDID}})
			}
			return nil, fmt.Errorf("erase iOS Simulator: %w", err)
		}
		state = "Shutdown"
	}
	if !strings.EqualFold(state, "Booted") {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "boot", lease.UDID}}); err != nil {
			return nil, fmt.Errorf("boot iOS Simulator: %w", err)
		}
	}
	bootCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.BootTimeoutMs)*time.Millisecond)
	defer cancel()
	if _, err := s.runner.Run(bootCtx, mobile.Command{Name: xcrun, Args: []string{"simctl", "bootstatus", lease.UDID, "-b"}}); err != nil {
		if lease.OwnedClone {
			_, _ = s.runner.Run(context.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "delete", lease.UDID}})
		}
		return nil, fmt.Errorf("wait for iOS Simulator boot: %w", err)
	}
	s.storeLease(lease)
	s.cleanupStack(ctx).Push("simulator:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.stopOwned(cleanupCtx, xcrun, lease)
		return err
	})
	return &SimulatorStartResponse{Lease: lease}, nil
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

func (s *service) simulatorStop(ctx *endly.Context, request *SimulatorStopRequest) (*SimulatorStopResponse, error) {
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	return s.stopOwned(ctx.Background(), xcrun, request.Lease)
}

func (s *service) stopOwned(ctx context.Context, xcrun string, lease DestinationLease) (*SimulatorStopResponse, error) {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	if !ok {
		s.mu.Unlock()
		return &SimulatorStopResponse{Warning: "lease already released or unknown"}, nil
	}
	if stored.Fence != lease.Fence || stored.UDID != lease.UDID {
		s.mu.Unlock()
		return nil, fmt.Errorf("Simulator lease fence mismatch")
	}
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	response := &SimulatorStopResponse{}
	if _, err := s.runner.Run(ctx, mobile.Command{Name: xcrun, Args: []string{"simctl", "shutdown", stored.UDID}}); err != nil {
		return nil, fmt.Errorf("shutdown iOS Simulator: %w", err)
	}
	response.Shutdown = true
	if stored.OwnedClone {
		if _, err := s.runner.Run(ctx, mobile.Command{Name: xcrun, Args: []string{"simctl", "delete", stored.UDID}}); err != nil {
			return response, fmt.Errorf("delete owned iOS Simulator: %w", err)
		}
		response.Deleted = true
	}
	return response, nil
}

func (s *service) storeLease(lease DestinationLease) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leases[lease.ID] = lease
}

type simctlDevice struct {
	UDID              string `json:"udid"`
	Name              string `json:"name"`
	State             string `json:"state"`
	IsAvailable       bool   `json:"isAvailable"`
	AvailabilityError string `json:"availabilityError"`
}

type simctlDevices struct {
	Devices map[string][]simctlDevice `json:"devices"`
}

func parseSimulators(source string) ([]IOSSimulator, error) {
	payload := simctlDevices{}
	if err := json.Unmarshal([]byte(source), &payload); err != nil {
		return nil, fmt.Errorf("decode simctl devices: %w", err)
	}
	result := []IOSSimulator{}
	for runtimeName, devices := range payload.Devices {
		for _, device := range devices {
			available := device.IsAvailable && device.AvailabilityError == ""
			if !available {
				continue
			}
			result = append(result, IOSSimulator{UDID: device.UDID, Name: device.Name, State: device.State, Runtime: runtimeName, IsAvailable: true})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Runtime == result[j].Runtime {
			return result[i].Name < result[j].Name
		}
		return result[i].Runtime < result[j].Runtime
	})
	return result, nil
}

func parseRuntimes(source string) []string {
	payload := struct {
		Runtimes []struct {
			Identifier        string `json:"identifier"`
			IsAvailable       bool   `json:"isAvailable"`
			AvailabilityError string `json:"availabilityError"`
		} `json:"runtimes"`
	}{}
	if json.Unmarshal([]byte(source), &payload) != nil {
		return nil
	}
	result := []string{}
	for _, runtime := range payload.Runtimes {
		if runtime.IsAvailable && runtime.AvailabilityError == "" {
			result = append(result, runtime.Identifier)
		}
	}
	sort.Strings(result)
	return result
}

func findSimulator(simulators []IOSSimulator, udid, name string) (IOSSimulator, bool) {
	var matched IOSSimulator
	count := 0
	for _, simulator := range simulators {
		if udid != "" && simulator.UDID != udid {
			continue
		}
		if name != "" && simulator.Name != name {
			continue
		}
		matched = simulator
		count++
	}
	return matched, count == 1
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}
