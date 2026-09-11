package ios

import (
	"context"
	"encoding/json"
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

const ServiceID = "ios"

type service struct {
	*endly.AbstractService
	runner     mobile.Runner
	mu         sync.Mutex
	leases     map[string]DestinationLease
	sessions   map[string]*iosSession
	servers    map[string]*iosServer
	captures   map[string]*iosCapture
	fs         afs.Service
	input      io.Reader
	output     io.Writer
	leaseStore *mobile.LeaseStore
}

type iosSession struct {
	handle         SessionHandle
	appium         *mobile.AppiumSession
	attached       bool
	ownsBackend    bool
	backendClosed  bool
	descriptorPath string
	mu             sync.Mutex
}

type iosCapture struct {
	handle CaptureHandle
	log    *mobile.LoggedProcess
	video  *mobile.SegmentedCapture
}

type iosServer struct {
	appium       *mobile.AppiumServer
	destination  DestinationLease
	appiumHome   string
	processLease *mobile.LeaseHandle
}

type contextCleanup struct{ stack *mobile.CleanupStack }

var cleanupKey = (*contextCleanup)(nil)

func newService(runner mobile.Runner) *service {
	result := &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		runner:          runner,
		leases:          map[string]DestinationLease{},
		sessions:        map[string]*iosSession{},
		servers:         map[string]*iosServer{},
		captures:        map[string]*iosCapture{},
		fs:              afs.New(),
		input:           os.Stdin,
		output:          os.Stdout,
		leaseStore:      mobile.NewLeaseStore(""),
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
		Action:           "device-list",
		RequestInfo:      &endly.ActionInfo{Description: "list paired Apple physical devices known to CoreDevice"},
		RequestProvider:  func() interface{} { return &DeviceListRequest{} },
		ResponseProvider: func() interface{} { return &DeviceListResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceList(ctx, request.(*DeviceListRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "device-lease",
		RequestInfo:      &endly.ActionInfo{Description: "lease one exact paired Apple physical device"},
		RequestProvider:  func() interface{} { return &DeviceLeaseRequest{} },
		ResponseProvider: func() interface{} { return &DeviceLeaseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceLease(ctx, request.(*DeviceLeaseRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "device-release",
		RequestInfo:      &endly.ActionInfo{Description: "release a physical-device lease without erasing or shutting down the device"},
		RequestProvider:  func() interface{} { return &DeviceReleaseRequest{} },
		ResponseProvider: func() interface{} { return &DeviceReleaseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceRelease(ctx, request.(*DeviceReleaseRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "destination-register",
		RequestInfo:      &endly.ActionInfo{Description: "register and fence an external/cloud iOS destination"},
		RequestProvider:  func() interface{} { return &DestinationRegisterRequest{} },
		ResponseProvider: func() interface{} { return &DestinationRegisterResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.destinationRegister(ctx, request.(*DestinationRegisterRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "destination-release",
		RequestInfo:      &endly.ActionInfo{Description: "release an external/cloud iOS destination registration"},
		RequestProvider:  func() interface{} { return &DestinationReleaseRequest{} },
		ResponseProvider: func() interface{} { return &DestinationReleaseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.destinationRelease(ctx, request.(*DestinationReleaseRequest))
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
		Action:           "attach",
		RequestInfo:      &endly.ActionInfo{Description: "attach to an existing iOS Appium session"},
		RequestProvider:  func() interface{} { return &AttachRequest{} },
		ResponseProvider: func() interface{} { return &AttachResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.attach(ctx, request.(*AttachRequest))
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
	s.Register(&endly.Route{
		Action:           "repl",
		RequestInfo:      &endly.ActionInfo{Description: "open a live iOS DSL and hierarchy inspector"},
		RequestProvider:  func() interface{} { return &REPLRequest{} },
		ResponseProvider: func() interface{} { return &REPLResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.repl(ctx, request.(*REPLRequest))
		},
	})
}

func (s *service) repl(ctx *endly.Context, request *REPLRequest) (*REPLResponse, error) {
	if request.Attach != nil {
		attached, err := s.attach(ctx, request.Attach)
		if err != nil {
			return nil, err
		}
		request.SessionID = attached.Session.ID
	}
	sessionID, session, err := s.resolveREPLSession(request.SessionID)
	if err != nil {
		return nil, err
	}
	if !session.attached {
		if err := s.validateLease(session.handle.Destination); err != nil {
			return nil, err
		}
	}
	prompt := request.Prompt
	if prompt == "" {
		prompt = "ios[" + sessionID + "]> "
	}
	result, err := mobile.RunREPL(ctx.Background(), s.input, s.output, mobile.REPLConfig{
		Prompt: prompt, MaxSourceBytes: request.MaxSourceBytes,
		MaxTreeNodes: request.MaxTreeNodes, FailOnError: request.FailOnError,
		HistoryPath: ctx.Expand(request.HistoryPath), MaxHistory: request.MaxHistory,
	}, mobile.REPLCallbacks{
		Execute: func(_ context.Context, command string) (*mobile.ExecutionResult, error) {
			var failureArtifacts *mobile.FailureArtifactOptions
			if request.ArtifactDirectory != "" {
				failureArtifacts = &mobile.FailureArtifactOptions{Directory: request.ArtifactDirectory}
			}
			response, err := s.run(ctx, &RunRequest{
				SessionID: sessionID, Commands: []interface{}{command},
				ActionTimeoutMs: request.ActionTimeoutMs, PollIntervalMs: request.PollIntervalMs,
				FailureArtifacts: failureArtifacts,
			})
			if response == nil {
				return nil, err
			}
			return &mobile.ExecutionResult{Data: response.Data, Steps: response.Steps, Validations: response.Validations, Failures: response.Failures}, err
		},
		Source: func(replCtx context.Context) (string, error) {
			session.mu.Lock()
			defer session.mu.Unlock()
			return session.appium.PageSource(replCtx)
		},
		Screenshot: func(context.Context) (*mobile.Evidence, error) {
			if request.ArtifactDirectory == "" {
				return nil, fmt.Errorf("ArtifactDirectory is required for :screenshot")
			}
			response, err := s.artifact(ctx, &ArtifactRequest{
				SessionID: sessionID, Directory: request.ArtifactDirectory, Screenshot: true,
			})
			if err != nil {
				return nil, err
			}
			return response.Artifacts[0], nil
		},
		Status: func(context.Context) interface{} {
			return map[string]interface{}{
				"platform": "ios", "sessionID": session.handle.ID,
				"backendSessionID": session.handle.BackendSessionID,
				"udid":             session.handle.Destination.UDID, "server": session.handle.Server.Endpoint,
				"attached": session.attached, "ownsBackend": session.ownsBackend,
			}
		},
		Close: func(replCtx context.Context) error {
			_, err := s.close(replCtx, &CloseRequest{SessionID: sessionID})
			return err
		},
	})
	return &REPLResponse{SessionID: sessionID, Result: result}, err
}

func (s *service) resolveREPLSession(requested string) (string, *iosSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if requested != "" {
		session, ok := s.sessions[requested]
		if !ok {
			return "", nil, fmt.Errorf("iOS session %q was not found", requested)
		}
		return requested, session, nil
	}
	if len(s.sessions) != 1 {
		return "", nil, fmt.Errorf("SessionID is required when %d iOS sessions are open", len(s.sessions))
	}
	for id, session := range s.sessions {
		return id, session, nil
	}
	return "", nil, fmt.Errorf("no iOS session is open")
}

func (s *service) launch(ctx *endly.Context, request *LaunchRequest) (*LaunchResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	if request.Destination.IsExternal() {
		return nil, fmt.Errorf("external iOS destinations are launched through Appium open/run")
	}
	if request.Destination.IsDevice() {
		return s.launchPhysicalApp(ctx, request)
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
	if request.Destination.IsExternal() {
		return nil, fmt.Errorf("external iOS destinations are terminated through Appium device.terminateApp")
	}
	if request.Destination.IsDevice() {
		return s.terminatePhysicalApp(ctx, request)
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
	if request.Destination.IsExternal() {
		return nil, fmt.Errorf("external iOS capture must be configured through the provider")
	}
	if request.Destination.IsDevice() {
		return nil, fmt.Errorf("physical-device log/video capture requires an Appium session or external collector")
	}
	xcrun, err := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if err != nil {
		return nil, err
	}
	captureID := "ios-capture-" + uuid.NewString()
	state := &iosCapture{}
	if request.LogPath != "" {
		args := []string{"simctl", "spawn", request.Destination.UDID, "log", "stream", "--style", "ndjson", "--level", "info"}
		if request.Predicate != "" {
			args = append(args, "--predicate", request.Predicate)
		}
		state.log, err = mobile.StartLoggedProcess(ctx.Background(), s.runner, mobile.Command{Name: xcrun, Args: args}, request.LogPath)
		if err != nil {
			return nil, err
		}
	}
	if request.Video {
		if err := os.MkdirAll(request.VideoDirectory, 0o700); err != nil {
			if state.log != nil {
				_ = state.log.Stop(context.Background())
			}
			return nil, err
		}
		segmentMs := request.SegmentMs
		if segmentMs <= 0 {
			segmentMs = 300_000
		}
		state.video, err = mobile.StartSegmentedCapture(time.Duration(segmentMs)*time.Millisecond,
			func(segmentCtx context.Context, index int) (*mobile.Process, error) {
				path := filepath.Join(request.VideoDirectory, fmt.Sprintf("segment-%04d.mp4", index))
				process, err := s.runner.Start(segmentCtx, mobile.Command{Name: xcrun, Args: []string{
					"simctl", "io", request.Destination.UDID, "recordVideo", "--codec=h264", "--force", path,
				}}, io.Discard, io.Discard)
				if err != nil {
					return nil, err
				}
				ready := time.NewTimer(time.Second)
				defer ready.Stop()
				select {
				case <-ready.C:
					return process, nil
				case processErr := <-process.Done:
					if processErr == nil {
						processErr = fmt.Errorf("recordVideo exited before initialization")
					}
					return nil, processErr
				case <-segmentCtx.Done():
					if process.Stop != nil {
						_ = process.Stop(context.Background())
					}
					return nil, segmentCtx.Err()
				}
			},
			func(_ context.Context, index int) (string, error) {
				path := filepath.Join(request.VideoDirectory, fmt.Sprintf("segment-%04d.mp4", index))
				if _, err := os.Stat(path); err != nil {
					return "", err
				}
				return path, nil
			},
		)
		if err != nil {
			if state.log != nil {
				_ = state.log.Stop(context.Background())
			}
			return nil, err
		}
	}
	pid := 0
	if state.log != nil {
		pid = state.log.PID
	}
	handle := CaptureHandle{ID: captureID, Destination: request.Destination, PID: pid, LogPath: request.LogPath}
	state.handle = handle
	s.mu.Lock()
	s.captures[handle.ID] = state
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("capture:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.captureStop(cleanupCtx, &CaptureStopRequest{Capture: handle})
		return err
	})
	return &CaptureStartResponse{Capture: handle}, nil
}

func (s *service) captureStop(ctx context.Context, request *CaptureStopRequest) (*CaptureStopResponse, error) {
	s.mu.Lock()
	capture, ok := s.captures[request.Capture.ID]
	s.mu.Unlock()
	if !ok {
		return &CaptureStopResponse{Warning: "capture already stopped or unknown"}, nil
	}
	if err := s.validateLease(request.Capture.Destination); err != nil {
		return nil, err
	}
	response := &CaptureStopResponse{Stopped: true, Artifacts: []*mobile.Evidence{}, Errors: []string{}}
	if capture.log != nil {
		if err := capture.log.Stop(ctx); err != nil && err != context.DeadlineExceeded {
			response.Errors = append(response.Errors, err.Error())
		}
		info, _ := os.Stat(capture.log.Path)
		size := 0
		if info != nil {
			size = int(info.Size())
		}
		logArtifact := &mobile.Evidence{Kind: "unifiedLog", URL: capture.log.Path, Size: size, Sensitive: true}
		response.Artifact = logArtifact
		response.Artifacts = append(response.Artifacts, logArtifact)
	}
	if capture.video != nil {
		paths, captureErrors := capture.video.Stop()
		response.Errors = append(response.Errors, captureErrors...)
		for _, path := range paths {
			info, _ := os.Stat(path)
			size := 0
			if info != nil {
				size = int(info.Size())
			}
			response.Artifacts = append(response.Artifacts, &mobile.Evidence{Kind: "video", URL: path, Size: size, Sensitive: true})
		}
	}
	s.mu.Lock()
	delete(s.captures, request.Capture.ID)
	s.mu.Unlock()
	if len(response.Errors) > 0 {
		return response, fmt.Errorf("capture cleanup: %s", strings.Join(response.Errors, "; "))
	}
	return response, nil
}

func (s *service) artifact(ctx *endly.Context, request *ArtifactRequest) (*ArtifactResponse, error) {
	if request.SessionID == "" {
		if request.Destination == nil {
			return nil, fmt.Errorf("Destination is required")
		}
		if err := s.validateLease(*request.Destination); err != nil {
			return nil, err
		}
		if request.Destination.IsExternal() {
			return nil, fmt.Errorf("external iOS screenshots require SessionID so Appium/WDA performs capture")
		}
		if request.Destination.IsDevice() {
			return nil, fmt.Errorf("physical-device screenshots require SessionID so Appium/WDA performs capture")
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
	if !session.attached {
		if err := s.validateLease(session.handle.Destination); err != nil {
			return nil, err
		}
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
	if request.Destination.IsExternal() {
		return nil, fmt.Errorf("external iOS apps must be uploaded by the provider and supplied as OpenRequest.AppReference")
	}
	if request.Destination.IsDevice() {
		return s.installPhysicalApp(ctx, request)
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
	if request.Destination.IsExternal() {
		return nil, fmt.Errorf("external iOS app removal is provider-managed")
	}
	if request.Destination.IsDevice() {
		return s.uninstallPhysicalApp(ctx, request)
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
	options := request.options()
	options.Init()
	var processLease *mobile.LeaseHandle
	var err error
	if options.Mode == "managed" {
		processLease, err = s.leaseStore.Acquire(ctx.Background(), fmt.Sprintf("appium:%s:%d", options.Address, options.Port))
		if err != nil {
			return nil, err
		}
	}
	server, err := mobile.StartAppium(ctx.Background(), s.runner, options)
	if err != nil {
		_ = s.leaseStore.Release(processLease)
		return nil, err
	}
	server.ID = "ios-server-" + uuid.NewString()
	handle := ServerHandle{ID: server.ID, Endpoint: server.Endpoint, Ownership: server.Ownership, PID: server.PID, LogPath: server.LogPath, ProcessLease: processLease}
	s.mu.Lock()
	s.servers[handle.ID] = &iosServer{appium: server, destination: request.Destination, appiumHome: request.AppiumHome, processLease: processLease}
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
	}
	s.mu.Unlock()
	if !ok {
		return &ServerStopResponse{Warning: "server already stopped or unknown"}, nil
	}
	if owned.processLease != nil {
		if request.Server.ProcessLease == nil || owned.processLease.Token != request.Server.ProcessLease.Token {
			return nil, fmt.Errorf("iOS persistent Appium lease token mismatch")
		}
		if err := s.leaseStore.Validate(owned.processLease); err != nil {
			return nil, err
		}
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
	s.mu.Lock()
	delete(s.servers, request.Server.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(owned.processLease); err != nil {
		return nil, err
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
	if err := validateWDACompatibility(request.WDA, request.Destination); err != nil {
		return nil, err
	}
	WDA, err := wdaCapabilities(ctx, request.WDA)
	if err != nil {
		return nil, err
	}
	for key, value := range WDA {
		capabilities[key] = value
	}
	defer delete(capabilities, "appium:keychainPassword")
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
	if request.AppReference != "" {
		capabilities["appium:app"] = ctx.Expand(request.AppReference)
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
	descriptorPath := ctx.Expand(request.DescriptorPath)
	if descriptorPath != "" {
		descriptor := &mobile.SessionDescriptor{
			Platform: "ios", SessionID: handle.ID, BackendSessionID: handle.BackendSessionID,
			Endpoint: ctx.Expand(handle.Server.Endpoint), TargetID: handle.Destination.UDID,
		}
		if err := mobile.WriteSessionDescriptor(descriptorPath, descriptor); err != nil {
			_ = appiumSession.Close(context.Background())
			return nil, err
		}
	}
	owned := &iosSession{
		handle: handle, appium: appiumSession, ownsBackend: true, descriptorPath: descriptorPath,
	}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		_ = appiumSession.Close(context.Background())
		_ = mobile.RemoveSessionDescriptor(descriptorPath)
		return nil, fmt.Errorf("iOS session %q already exists", handle.ID)
	}
	s.sessions[handle.ID] = owned
	s.mu.Unlock()
	if !request.KeepSession {
		s.cleanupStack(ctx).Push("session:"+handle.ID, func(cleanupCtx context.Context) error {
			_, err := s.close(cleanupCtx, &CloseRequest{SessionID: handle.ID})
			return err
		})
	}
	return &OpenResponse{Session: handle}, nil
}

func (s *service) attach(ctx *endly.Context, request *AttachRequest) (*AttachResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	descriptorPath := ctx.Expand(request.DescriptorPath)
	descriptor := &mobile.SessionDescriptor{
		Platform: "ios", SessionID: request.SessionID, BackendSessionID: request.BackendSessionID,
		Endpoint: ctx.Expand(request.ServerURL), TargetID: request.TargetID,
	}
	if descriptorPath != "" {
		loaded, err := mobile.ReadSessionDescriptor(descriptorPath, "ios")
		if err != nil {
			return nil, err
		}
		descriptor = loaded
		if request.SessionID != "" {
			descriptor.SessionID = request.SessionID
		}
	} else {
		descriptor.Init()
		if err := descriptor.Validate("ios"); err != nil {
			return nil, err
		}
	}
	if descriptor.SessionID == "" {
		descriptor.SessionID = "ios-attached-" + descriptor.BackendSessionID
	}
	client, err := mobile.NewAppiumClient(descriptor.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	appiumSession, err := client.AttachSession(ctx.Background(), descriptor.BackendSessionID)
	if err != nil {
		return nil, err
	}
	handle := SessionHandle{
		ID: descriptor.SessionID, BackendSessionID: descriptor.BackendSessionID,
		Destination: DestinationLease{UDID: descriptor.TargetID},
		Server:      ServerHandle{Endpoint: descriptor.Endpoint, Ownership: "external"},
	}
	attached := &iosSession{
		handle: handle, appium: appiumSession, attached: true,
		ownsBackend: request.TakeOwnership, descriptorPath: descriptorPath,
	}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("iOS session %q already exists", handle.ID)
	}
	s.sessions[handle.ID] = attached
	s.mu.Unlock()
	s.cleanupStack(ctx).Push("attached-session:"+handle.ID, func(cleanupCtx context.Context) error {
		_, err := s.close(cleanupCtx, &CloseRequest{SessionID: handle.ID})
		return err
	})
	return &AttachResponse{Session: handle}, nil
}

func (s *service) run(ctx *endly.Context, request *RunRequest) (*RunResponse, error) {
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("iOS session %q was not found", request.SessionID)
	}
	if !session.attached {
		if err := s.validateLease(session.handle.Destination); err != nil {
			return nil, err
		}
	}
	commands := make([]interface{}, len(request.Commands))
	state := ctx.State()
	for i, command := range request.Commands {
		if text, ok := command.(string); ok {
			commands[i] = ctx.Expand(text)
		} else {
			commands[i] = state.Expand(command)
		}
	}
	executor := &mobile.Executor{
		Session:       session.appium,
		Resolve:       iosLocatorResolver,
		ExecuteDevice: executeIOSDevice,
		ActionTimeout: time.Duration(request.ActionTimeoutMs) * time.Millisecond,
		PollInterval:  time.Duration(request.PollIntervalMs) * time.Millisecond,
	}
	session.mu.Lock()
	result, err := executor.RunAny(ctx.Background(), commands)
	session.mu.Unlock()
	if result == nil {
		return nil, err
	}
	response := &RunResponse{Data: result.Data, Steps: result.Steps, Validations: result.Validations, Failures: []*mobile.FailureEvidence{}}
	reason := ""
	if err != nil {
		reason = err.Error()
	} else if validationReason, failed := mobile.ValidationFailureReason(result.Validations); failed {
		reason = validationReason
	}
	if reason != "" && request.FailureArtifacts != nil {
		options := *request.FailureArtifacts
		options.Directory = ctx.Expand(options.Directory)
		files, collectionErrors := s.failureCaptureFiles(ctx.Background(), session)
		options.Files = append(options.Files, files...)
		options.CollectionErrors = append(options.CollectionErrors, collectionErrors...)
		prefix := "ios-failure-" + safeArtifactPart(request.SessionID) + "-" + uuid.NewString()
		session.mu.Lock()
		failure := mobile.CaptureAppiumFailure(ctx.Background(), s.fs, session.appium, prefix, reason, &options)
		session.mu.Unlock()
		if failure != nil {
			response.Failures = append(response.Failures, failure)
		}
	}
	return response, err
}

func (s *service) failureCaptureFiles(ctx context.Context, session *iosSession) ([]mobile.FailureArtifactFile, []string) {
	s.mu.Lock()
	captures := make([]*iosCapture, 0)
	for _, capture := range s.captures {
		if capture != nil && capture.handle.Destination.ID != "" && capture.handle.Destination.ID == session.handle.Destination.ID {
			captures = append(captures, capture)
		}
	}
	s.mu.Unlock()
	files := []mobile.FailureArtifactFile{}
	errors := []string{}
	for _, capture := range captures {
		if capture.log != nil && capture.log.Path != "" {
			files = append(files, mobile.FailureArtifactFile{Path: capture.log.Path, Kind: "unifiedLog", MaxBytes: 2 << 20, Tail: true})
		}
		if capture.video != nil {
			checkpointCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			paths, checkpointErrors := capture.video.Checkpoint(checkpointCtx)
			cancel()
			for _, checkpointError := range checkpointErrors {
				errors = append(errors, "video checkpoint: "+checkpointError)
			}
			if len(paths) > 0 {
				files = append(files, mobile.FailureArtifactFile{Path: paths[len(paths)-1], Kind: "video", MaxBytes: 100 << 20})
			}
		}
	}
	return files, errors
}

func (s *service) close(ctx context.Context, request *CloseRequest) (*CloseResponse, error) {
	s.mu.Lock()
	session, ok := s.sessions[request.SessionID]
	s.mu.Unlock()
	if !ok {
		return &CloseResponse{Warning: "session already closed or unknown"}, nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.ownsBackend {
		s.mu.Lock()
		delete(s.sessions, request.SessionID)
		s.mu.Unlock()
		return &CloseResponse{Closed: true, Warning: "detached from external Appium session; backend session remains open"}, nil
	}
	if !session.backendClosed {
		if err := session.appium.Close(ctx); err != nil {
			return nil, err
		}
		session.backendClosed = true
	}
	if err := mobile.RemoveSessionDescriptor(session.descriptorPath); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.sessions[request.SessionID] == session {
		delete(s.sessions, request.SessionID)
	}
	s.mu.Unlock()
	return &CloseResponse{Closed: true}, nil
}

func (s *service) validateLease(lease DestinationLease) error {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("iOS destination lease %q was not found", lease.ID)
	}
	if stored.Fence != lease.Fence || stored.UDID != lease.UDID {
		return fmt.Errorf("iOS destination lease fence mismatch")
	}
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return fmt.Errorf("iOS persistent destination lease token mismatch")
		}
		if err := s.leaseStore.Refresh(stored.ProcessLease); err != nil {
			return fmt.Errorf("validate iOS persistent destination lease: %w", err)
		}
	}
	return nil
}

func (s *service) validateServer(handle ServerHandle) error {
	s.mu.Lock()
	server, ok := s.servers[handle.ID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("iOS Appium server %q was not found", handle.ID)
	}
	if server.appium.Endpoint != handle.Endpoint || server.appium.Ownership != handle.Ownership {
		return fmt.Errorf("iOS Appium server handle mismatch")
	}
	if server.processLease != nil {
		if handle.ProcessLease == nil || server.processLease.Token != handle.ProcessLease.Token {
			return fmt.Errorf("iOS persistent Appium lease token mismatch")
		}
		if err := s.leaseStore.Refresh(server.processLease); err != nil {
			return fmt.Errorf("validate iOS persistent Appium lease: %w", err)
		}
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
	case "locator":
		if len(call.Args) != 2 {
			return mobile.Locator{}, true, fmt.Errorf("locator requires strategy and value")
		}
		strategy, err := stringCallArg(call, 0)
		if err != nil {
			return mobile.Locator{}, true, err
		}
		value, err = stringCallArg(call, 1)
		if err != nil {
			return mobile.Locator{}, true, err
		}
		using, ok := iosLocatorStrategy(strategy)
		if !ok {
			return mobile.Locator{}, true, fmt.Errorf("unsupported iOS locator strategy %q", strategy)
		}
		return mobile.Locator{Using: using, Value: value}, true, nil
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

func iosLocatorStrategy(strategy string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "accessibilityid", "accessibility id":
		return "accessibility id", true
	case "name":
		return "name", true
	case "class", "classname", "class name":
		return "class name", true
	case "predicate", "-ios predicate string":
		return "-ios predicate string", true
	case "classchain", "-ios class chain":
		return "-ios class chain", true
	case "xpath":
		return "xpath", true
	case "css", "css selector":
		return "css selector", true
	default:
		return "", false
	}
}

func executeIOSDevice(ctx context.Context, session *mobile.AppiumSession, call mobile.Call) (interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "home":
		return session.Execute(ctx, "mobile: pressButton", map[string]interface{}{"name": "home"})
	case "hidekeyboard":
		return nil, session.HideKeyboard(ctx)
	case "orientation":
		return session.Orientation(ctx)
	case "rotate":
		orientation, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		orientation = strings.ToUpper(orientation)
		if orientation != "PORTRAIT" && orientation != "LANDSCAPE" {
			return nil, fmt.Errorf("orientation must be PORTRAIT or LANDSCAPE")
		}
		return orientation, session.SetOrientation(ctx, orientation)
	case "contexts":
		return session.Contexts(ctx)
	case "context":
		if len(call.Args) == 0 {
			return session.CurrentContext(ctx)
		}
		name, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return name, session.SetContext(ctx, name)
	case "setlocation":
		latitude, err := floatCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		longitude, err := floatCallArg(call, 1)
		if err != nil {
			return nil, err
		}
		altitude := optionalFloatCallArg(call, 2, 0)
		return nil, session.SetLocation(ctx, latitude, longitude, altitude)
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
	case "acceptalert":
		return nil, session.AcceptAlert(ctx)
	case "dismissalert":
		return nil, session.DismissAlert(ctx)
	case "alerttext":
		return session.AlertText(ctx)
	case "appearance":
		appearance, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: setAppearance", map[string]interface{}{"style": appearance})
	case "setpermission":
		bundleID, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		service, err := stringCallArg(call, 1)
		if err != nil {
			return nil, err
		}
		state, err := stringCallArg(call, 2)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: setPermission", map[string]interface{}{
			"bundleId": bundleID, "access": map[string]string{service: state},
		})
	case "enrollbiometric":
		enrolled, err := boolCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: enrollBiometric", map[string]interface{}{"isEnabled": enrolled})
	case "matchbiometric":
		kind, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		match, err := boolCallArg(call, 1)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: sendBiometricMatch", map[string]interface{}{"type": kind, "match": match})
	case "activateapp", "terminateapp":
		bundleID, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		script := "mobile: activateApp"
		if strings.EqualFold(call.Name, "terminateApp") {
			script = "mobile: terminateApp"
		}
		return session.Execute(ctx, script, map[string]interface{}{"bundleId": bundleID})
	case "backgroundapp":
		seconds, err := intCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: backgroundApp", map[string]interface{}{"seconds": seconds})
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

func intCallArg(call mobile.Call, index int) (int, error) {
	if index >= len(call.Args) {
		return 0, fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	switch value := call.Args[index].(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		return int(value), nil
	default:
		return 0, fmt.Errorf("%s argument %d must be an integer", call.Name, index+1)
	}
}

func floatCallArg(call mobile.Call, index int) (float64, error) {
	if index >= len(call.Args) {
		return 0, fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	switch value := call.Args[index].(type) {
	case float64:
		return value, nil
	case int64:
		return float64(value), nil
	case int:
		return float64(value), nil
	default:
		return 0, fmt.Errorf("%s argument %d must be numeric", call.Name, index+1)
	}
}

func optionalFloatCallArg(call mobile.Call, index int, fallback float64) float64 {
	value, err := floatCallArg(call, index)
	if err != nil {
		return fallback
	}
	return value
}

func boolCallArg(call mobile.Call, index int) (bool, error) {
	if index >= len(call.Args) {
		return false, fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	value, ok := call.Args[index].(bool)
	if !ok {
		return false, fmt.Errorf("%s argument %d must be boolean", call.Name, index+1)
	}
	return value, nil
}

func escapePredicate(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`)
}

func isProtectedIOSCapability(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "platformname", "appium:automationname", "appium:udid", "appium:bundleid", "appium:app", "udid", "automationname", "bundleid", "app",
		"appium:useprebuiltwda", "useprebuiltwda", "appium:usepreinstalledwda", "usepreinstalledwda",
		"appium:prebuiltwdapath", "prebuiltwdapath", "appium:webdriveragenturl", "webdriveragenturl",
		"appium:deriveddatapath", "deriveddatapath", "appium:updatedwdabundleid", "updatedwdabundleid",
		"appium:updatedwdabundleidsuffix", "updatedwdabundleidsuffix", "appium:xcodeorgid", "xcodeorgid",
		"appium:xcodesigningid", "xcodesigningid", "appium:xcodeconfigfile", "xcodeconfigfile",
		"appium:keychainpath", "keychainpath", "appium:keychainpassword", "keychainpassword",
		"appium:wdalocalport", "wdalocalport", "appium:mjpegserverport", "mjpegserverport",
		"appium:usenewwda", "usenewwda", "appium:prebuildwda", "prebuildwda",
		"appium:wdalaunchtimeout", "wdalaunchtimeout", "appium:wdaconnectiontimeout", "wdaconnectiontimeout",
		"appium:wdastartupretries", "wdastartupretries", "appium:wdastartupretryinterval", "wdastartupretryinterval":
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
	deviceCtlCheck := s.toolCheck(ctx.Background(), "devicectl", xcrun, xcrunErr, []string{"devicectl", "--version"}, "install full Xcode with CoreDevice/devicectl support")
	response.Checks = append(response.Checks, deviceCtlCheck)
	if deviceCtlCheck.Status == "ok" {
		if devices, err := s.listPhysicalDevices(ctx.Background(), 30_000); err == nil {
			response.Devices = devices
		}
	}
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
	resourceKey := "ios:simulator:create:" + request.DeviceType + ":" + request.Runtime
	if request.UDID != "" {
		resourceKey = "ios:simulator:udid:" + request.UDID
	} else if request.BaseName != "" {
		resourceKey = "ios:simulator:name:" + request.BaseName
		if request.CloneName != "" {
			resourceKey = "ios:simulator:name:" + request.CloneName
		}
	}
	processLease, err := s.leaseStore.Acquire(ctx.Background(), resourceKey)
	if err != nil {
		return nil, err
	}
	leaseCommitted := false
	defer func() {
		if !leaseCommitted {
			_ = s.leaseStore.Release(processLease)
		}
	}()
	list, err := s.runner.Run(ctx.Background(), mobile.Command{Name: xcrun, Args: []string{"simctl", "list", "devices", "available", "--json"}})
	if err != nil {
		return nil, fmt.Errorf("list iOS Simulators: %w", err)
	}
	simulators, err := parseSimulators(list.Stdout)
	if err != nil {
		return nil, err
	}
	lease := DestinationLease{ID: uuid.NewString(), Fence: processLease.Fence, Kind: "simulator", ProcessLease: processLease, PreserveOnRelease: request.KeepBooted}
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
	leaseCommitted = true
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
	if lease.IsDevice() {
		return nil, fmt.Errorf("physical-device leases must be released with ios:device-release")
	}
	if lease.IsExternal() {
		return nil, fmt.Errorf("external destination leases must be released with ios:destination-release")
	}
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
	s.mu.Unlock()
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return nil, fmt.Errorf("iOS persistent destination lease token mismatch")
		}
		if err := s.leaseStore.Validate(stored.ProcessLease); err != nil {
			return nil, err
		}
	}
	response := &SimulatorStopResponse{}
	if stored.PreserveOnRelease {
		s.mu.Lock()
		delete(s.leases, lease.ID)
		s.mu.Unlock()
		if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
			return response, err
		}
		response.Warning = "attached Simulator lease was released and left booted"
		return response, nil
	}
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
	s.mu.Lock()
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
		return response, err
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
