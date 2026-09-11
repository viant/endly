package android

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
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
	runner       mobile.Runner
	mu           sync.Mutex
	leases       map[string]DeviceLease
	sessions     map[string]*androidSession
	servers      map[string]*mobile.AppiumServer
	serverLeases map[string]*mobile.LeaseHandle
	captures     map[string]*androidCapture
	fs           afs.Service
	input        io.Reader
	output       io.Writer
	leaseStore   *mobile.LeaseStore
}

type androidSession struct {
	handle         SessionHandle
	appium         *mobile.AppiumSession
	testIDStrategy string
	attached       bool
	ownsBackend    bool
	backendClosed  bool
	descriptorPath string
	mu             sync.Mutex
}

type androidCapture struct {
	handle CaptureHandle
	log    *mobile.LoggedProcess
	video  *mobile.SegmentedCapture
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
		serverLeases:    map[string]*mobile.LeaseHandle{},
		captures:        map[string]*androidCapture{},
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
		Action:           "device-register",
		RequestInfo:      &endly.ActionInfo{Description: "register and fence an external/cloud Android destination"},
		RequestProvider:  func() interface{} { return &DeviceRegisterRequest{} },
		ResponseProvider: func() interface{} { return &DeviceRegisterResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceRegister(ctx, request.(*DeviceRegisterRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "device-release",
		RequestInfo:      &endly.ActionInfo{Description: "release an external/cloud Android destination registration"},
		RequestProvider:  func() interface{} { return &DeviceReleaseRequest{} },
		ResponseProvider: func() interface{} { return &DeviceReleaseResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.deviceRelease(ctx, request.(*DeviceReleaseRequest))
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
		Action:           "attach",
		RequestInfo:      &endly.ActionInfo{Description: "attach to an existing Android Appium session"},
		RequestProvider:  func() interface{} { return &AttachRequest{} },
		ResponseProvider: func() interface{} { return &AttachResponse{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return s.attach(ctx, request.(*AttachRequest))
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
	s.Register(&endly.Route{
		Action:           "repl",
		RequestInfo:      &endly.ActionInfo{Description: "open a live Android DSL and hierarchy inspector"},
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
		if err := s.validateLease(session.handle.Lease); err != nil {
			return nil, err
		}
	}
	prompt := request.Prompt
	if prompt == "" {
		prompt = "android[" + sessionID + "]> "
	}
	result, err := mobile.RunREPL(ctx.Background(), s.input, s.output, mobile.REPLConfig{
		Prompt: prompt, MaxSourceBytes: request.MaxSourceBytes,
		MaxTreeNodes: request.MaxTreeNodes, FailOnError: request.FailOnError,
		HistoryPath: ctx.Expand(request.HistoryPath), MaxHistory: request.MaxHistory,
	}, mobile.REPLCallbacks{
		Execute: func(replCtx context.Context, command string) (*mobile.ExecutionResult, error) {
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
				"platform": "android", "sessionID": session.handle.ID,
				"backendSessionID": session.handle.BackendSessionID,
				"serial":           session.handle.Lease.Serial, "server": session.handle.Server.Endpoint,
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

func (s *service) resolveREPLSession(requested string) (string, *androidSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if requested != "" {
		session, ok := s.sessions[requested]
		if !ok {
			return "", nil, fmt.Errorf("Android session %q was not found", requested)
		}
		return requested, session, nil
	}
	if len(s.sessions) != 1 {
		return "", nil, fmt.Errorf("SessionID is required when %d Android sessions are open", len(s.sessions))
	}
	for id, session := range s.sessions {
		return id, session, nil
	}
	return "", nil, fmt.Errorf("no Android session is open")
}

func (s *service) launch(ctx *endly.Context, request *LaunchRequest) (*LaunchResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	if request.Lease.External {
		return nil, fmt.Errorf("external Android destinations are launched through Appium open/run")
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
	if request.Lease.External {
		return nil, fmt.Errorf("external Android destinations are terminated through Appium device.terminateApp")
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
	if request.Lease.External {
		return nil, fmt.Errorf("external Android capture must be configured through the provider")
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	captureID := "android-capture-" + uuid.NewString()
	state := &androidCapture{}
	if request.Clear && request.LogPath != "" {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "logcat", "-c"}}); err != nil {
			return nil, fmt.Errorf("clear logcat: %w", err)
		}
	}
	if request.LogPath != "" {
		args := []string{"-s", request.Lease.Serial, "logcat", "-v", "threadtime"}
		if request.Package != "" {
			pid, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "pidof", "-s", request.Package}})
			if err != nil || strings.TrimSpace(pid.Stdout) == "" {
				return nil, fmt.Errorf("resolve Android package %s process ID", request.Package)
			}
			args = append(args, "--pid", strings.TrimSpace(pid.Stdout))
		}
		state.log, err = mobile.StartLoggedProcess(ctx.Background(), s.runner, mobile.Command{Name: adb, Args: args}, request.LogPath)
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
		if segmentMs <= 0 || segmentMs > 170_000 {
			segmentMs = 170_000
		}
		remotePrefix := "/sdcard/" + captureID
		state.video, err = mobile.StartSegmentedCapture(time.Duration(segmentMs)*time.Millisecond,
			func(segmentCtx context.Context, index int) (*mobile.Process, error) {
				remote := fmt.Sprintf("%s-%04d.mp4", remotePrefix, index)
				seconds := max(1, int(math.Ceil(float64(segmentMs)/1000)))
				return s.runner.Start(segmentCtx, mobile.Command{Name: adb, Args: []string{
					"-s", request.Lease.Serial, "shell", "screenrecord", "--time-limit", strconv.Itoa(seconds), remote,
				}}, io.Discard, io.Discard)
			},
			func(segmentCtx context.Context, index int) (string, error) {
				remote := fmt.Sprintf("%s-%04d.mp4", remotePrefix, index)
				local := filepath.Join(request.VideoDirectory, fmt.Sprintf("segment-%04d.mp4", index))
				result, pullErr := s.runner.Run(segmentCtx, mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "pull", remote, local}})
				_, _ = s.runner.Run(segmentCtx, mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "rm", "-f", remote}})
				if pullErr != nil {
					return "", fmt.Errorf("pull Android video: %s: %w", strings.TrimSpace(result.Stderr), pullErr)
				}
				return local, nil
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
	handle := CaptureHandle{ID: captureID, Lease: request.Lease, PID: pid, LogPath: request.LogPath}
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
	if err := s.validateLease(request.Capture.Lease); err != nil {
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
		logArtifact := &mobile.Evidence{Kind: "logcat", URL: capture.log.Path, Size: size, Sensitive: true}
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
		if request.Lease == nil {
			return nil, fmt.Errorf("Lease is required")
		}
		if err := s.validateLease(*request.Lease); err != nil {
			return nil, err
		}
		if request.Lease.External {
			return nil, fmt.Errorf("external Android screenshots require SessionID so Appium performs capture")
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
	if !session.attached {
		if err := s.validateLease(session.handle.Lease); err != nil {
			return nil, err
		}
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
	if request.Lease.External {
		return nil, fmt.Errorf("external Android apps must be uploaded by the provider and supplied as OpenRequest.AppReference")
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if request.State == "freshInstall" {
		installed, listErr := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "pm", "list", "packages", "--user", "0", request.Package}})
		if listErr != nil {
			return nil, fmt.Errorf("check existing Android package %s: %w", request.Package, listErr)
		}
		if strings.TrimSpace(installed.Stdout) != "" {
			if result, uninstallErr := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "uninstall", request.Package}}); uninstallErr != nil {
				return nil, fmt.Errorf("remove existing Android package %s: %s: %w", request.Package, strings.TrimSpace(result.Stdout+result.Stderr), uninstallErr)
			}
		}
	}
	artifacts, err := s.installAndroidArtifacts(ctx.Background(), adb, request)
	if err != nil {
		return nil, err
	}
	if request.State == "cleanData" {
		if _, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: []string{"-s", request.Lease.Serial, "shell", "pm", "clear", request.Package}}); err != nil {
			return nil, fmt.Errorf("clear Android package %s: %w", request.Package, err)
		}
	}
	return &InstallResponse{Installed: true, Package: request.Package, APKPath: request.APKPath, Artifacts: artifacts, State: request.State}, nil
}

func (s *service) installAndroidArtifacts(ctx context.Context, adb string, request *InstallRequest) ([]string, error) {
	installFlags := []string{}
	if request.State != "freshInstall" {
		installFlags = append(installFlags, "-r")
	}
	if request.GrantAll {
		installFlags = append(installFlags, "-g")
	}
	if request.AllowTest {
		installFlags = append(installFlags, "-t")
	}
	if request.AllowDowngrade {
		installFlags = append(installFlags, "-d")
	}
	if request.APKPath != "" {
		args := append([]string{"-s", request.Lease.Serial, "install"}, installFlags...)
		args = append(args, request.APKPath)
		if result, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: args}); err != nil {
			return nil, fmt.Errorf("install Android package %s: %s: %w", request.Package, strings.TrimSpace(result.Stderr), err)
		}
		return []string{request.APKPath}, nil
	}
	if len(request.APKPaths) > 0 {
		args := append([]string{"-s", request.Lease.Serial, "install-multiple"}, installFlags...)
		args = append(args, request.APKPaths...)
		if result, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: args}); err != nil {
			return nil, fmt.Errorf("install Android split package %s: %s: %w", request.Package, strings.TrimSpace(result.Stderr), err)
		}
		return append([]string(nil), request.APKPaths...), nil
	}
	apksPath := request.APKSPath
	if request.AABPath != "" {
		file, err := os.CreateTemp("", "endly-bundle-*.apks")
		if err != nil {
			return nil, err
		}
		apksPath = file.Name()
		_ = file.Close()
		_ = os.Remove(apksPath)
		defer os.Remove(apksPath)
		args := []string{"build-apks", "--bundle=" + request.AABPath, "--output=" + apksPath, "--overwrite"}
		if request.Signing != nil {
			args = append(args,
				"--ks="+request.Signing.KeystorePath,
				"--ks-key-alias="+request.Signing.KeyAlias,
				"--ks-pass=file:"+request.Signing.StorePasswordFile,
			)
			if request.Signing.KeyPasswordFile != "" {
				args = append(args, "--key-pass=file:"+request.Signing.KeyPasswordFile)
			}
		}
		if result, err := s.runBundletool(ctx, request, args); err != nil {
			return nil, fmt.Errorf("build APK set from AAB: %s: %w", strings.TrimSpace(result.Stdout+result.Stderr), err)
		}
	}
	if result, err := s.runBundletool(ctx, request, []string{"install-apks", "--apks=" + apksPath, "--device-id=" + request.Lease.Serial, "--adb=" + adb}); err != nil {
		return nil, fmt.Errorf("install Android APK set: %s: %w", strings.TrimSpace(result.Stdout+result.Stderr), err)
	}
	if request.AABPath != "" {
		return []string{request.AABPath}, nil
	}
	return []string{request.APKSPath}, nil
}

func (s *service) runBundletool(ctx context.Context, request *InstallRequest, args []string) (mobile.Result, error) {
	bundletool := request.BundletoolPath
	if strings.HasSuffix(strings.ToLower(bundletool), ".jar") {
		if info, err := os.Stat(bundletool); err != nil || info.IsDir() {
			return mobile.Result{}, fmt.Errorf("bundletool jar %q was not found", bundletool)
		}
		java := request.JavaPath
		if java == "" {
			if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
				java = filepath.Join(javaHome, "bin", "java")
			}
		}
		resolved, err := mobile.ResolveExecutable("java", java)
		if err != nil {
			return mobile.Result{}, err
		}
		return s.runner.Run(ctx, mobile.Command{Name: resolved, Args: append([]string{"-jar", bundletool}, args...)})
	}
	resolved, err := mobile.ResolveExecutable(bundletool)
	if err != nil {
		return mobile.Result{}, err
	}
	return s.runner.Run(ctx, mobile.Command{Name: resolved, Args: args})
}

func (s *service) uninstall(ctx *endly.Context, request *UninstallRequest) (*UninstallResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	if request.Lease.External {
		return nil, fmt.Errorf("external Android app removal is provider-managed")
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
	server.ID = "android-server-" + uuid.NewString()
	handle := ServerHandle{ID: server.ID, Endpoint: server.Endpoint, Ownership: server.Ownership, PID: server.PID, LogPath: server.LogPath, ProcessLease: processLease}
	s.mu.Lock()
	s.servers[handle.ID] = server
	s.serverLeases[handle.ID] = processLease
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
	processLease := s.serverLeases[request.Server.ID]
	if ok {
		if server.Endpoint != request.Server.Endpoint || server.Ownership != request.Server.Ownership {
			s.mu.Unlock()
			return nil, fmt.Errorf("Appium server handle mismatch")
		}
	}
	s.mu.Unlock()
	if !ok {
		return &ServerStopResponse{Warning: "server already stopped or unknown"}, nil
	}
	if processLease != nil {
		if request.Server.ProcessLease == nil || processLease.Token != request.Server.ProcessLease.Token {
			return nil, fmt.Errorf("Android persistent Appium lease token mismatch")
		}
		if err := s.leaseStore.Validate(processLease); err != nil {
			return nil, err
		}
	}
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := server.Stop(stopCtx); err != nil && err != context.DeadlineExceeded {
		return nil, err
	}
	s.mu.Lock()
	delete(s.servers, request.Server.ID)
	delete(s.serverLeases, request.Server.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(processLease); err != nil {
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
	if request.Package != "" {
		capabilities["appium:appPackage"] = request.Package
	}
	if request.AppReference != "" {
		capabilities["appium:app"] = ctx.Expand(request.AppReference)
	}
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
	descriptorPath := ctx.Expand(request.DescriptorPath)
	if descriptorPath != "" {
		descriptor := &mobile.SessionDescriptor{
			Platform: "android", SessionID: handle.ID, BackendSessionID: handle.BackendSessionID,
			Endpoint: ctx.Expand(handle.Server.Endpoint), TargetID: handle.Lease.Serial, TestIDStrategy: request.TestIDStrategy,
		}
		if err := mobile.WriteSessionDescriptor(descriptorPath, descriptor); err != nil {
			_ = appiumSession.Close(context.Background())
			return nil, err
		}
	}
	owned := &androidSession{
		handle: handle, appium: appiumSession, testIDStrategy: request.TestIDStrategy,
		ownsBackend: true, descriptorPath: descriptorPath,
	}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		_ = appiumSession.Close(context.Background())
		_ = mobile.RemoveSessionDescriptor(descriptorPath)
		return nil, fmt.Errorf("Android session %q already exists", handle.ID)
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
		Platform: "android", SessionID: request.SessionID, BackendSessionID: request.BackendSessionID,
		Endpoint: ctx.Expand(request.ServerURL), TargetID: request.TargetID, TestIDStrategy: request.TestIDStrategy,
	}
	if descriptorPath != "" {
		loaded, err := mobile.ReadSessionDescriptor(descriptorPath, "android")
		if err != nil {
			return nil, err
		}
		descriptor = loaded
		if request.SessionID != "" {
			descriptor.SessionID = request.SessionID
		}
	} else {
		descriptor.Init()
		if err := descriptor.Validate("android"); err != nil {
			return nil, err
		}
	}
	switch descriptor.TestIDStrategy {
	case "accessibilityId", "resourceId", "composeResourceId":
	default:
		return nil, fmt.Errorf("attached Android session requires a valid TestIDStrategy")
	}
	if descriptor.SessionID == "" {
		descriptor.SessionID = "android-attached-" + descriptor.BackendSessionID
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
		Lease:  DeviceLease{Serial: descriptor.TargetID},
		Server: ServerHandle{Endpoint: descriptor.Endpoint, Ownership: "external"},
	}
	attached := &androidSession{
		handle: handle, appium: appiumSession, testIDStrategy: descriptor.TestIDStrategy,
		attached: true, ownsBackend: request.TakeOwnership, descriptorPath: descriptorPath,
	}
	s.mu.Lock()
	if _, exists := s.sessions[handle.ID]; exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("Android session %q already exists", handle.ID)
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
		return nil, fmt.Errorf("Android session %q was not found", request.SessionID)
	}
	if !session.attached {
		if err := s.validateLease(session.handle.Lease); err != nil {
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
		Resolve:       androidLocatorResolver(session.testIDStrategy),
		ExecuteDevice: executeAndroidDevice,
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
		prefix := "android-failure-" + safeArtifactPart(request.SessionID) + "-" + uuid.NewString()
		session.mu.Lock()
		failure := mobile.CaptureAppiumFailure(ctx.Background(), s.fs, session.appium, prefix, reason, &options)
		session.mu.Unlock()
		if failure != nil {
			response.Failures = append(response.Failures, failure)
		}
	}
	return response, err
}

func (s *service) failureCaptureFiles(ctx context.Context, session *androidSession) ([]mobile.FailureArtifactFile, []string) {
	s.mu.Lock()
	captures := make([]*androidCapture, 0)
	for _, capture := range s.captures {
		if capture != nil && capture.handle.Lease.ID != "" && capture.handle.Lease.ID == session.handle.Lease.ID {
			captures = append(captures, capture)
		}
	}
	s.mu.Unlock()
	files := []mobile.FailureArtifactFile{}
	errors := []string{}
	for _, capture := range captures {
		if capture.log != nil && capture.log.Path != "" {
			files = append(files, mobile.FailureArtifactFile{Path: capture.log.Path, Kind: "logcat", MaxBytes: 2 << 20, Tail: true})
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

func (s *service) validateLease(lease DeviceLease) error {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("Android device lease %q was not found", lease.ID)
	}
	if stored.Fence != lease.Fence || stored.Serial != lease.Serial {
		return fmt.Errorf("Android device lease fence mismatch")
	}
	if stored.SerialLease != nil {
		if err := s.leaseStore.Validate(stored.SerialLease); err != nil {
			return err
		}
	}
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return fmt.Errorf("Android persistent device lease token mismatch")
		}
		if err := s.leaseStore.Refresh(stored.ProcessLease); err != nil {
			return fmt.Errorf("validate Android persistent device lease: %w", err)
		}
	}
	return nil
}

func (s *service) validateServer(handle ServerHandle) error {
	s.mu.Lock()
	server, ok := s.servers[handle.ID]
	processLease := s.serverLeases[handle.ID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("Android Appium server %q was not found", handle.ID)
	}
	if server.Endpoint != handle.Endpoint || server.Ownership != handle.Ownership {
		return fmt.Errorf("Android Appium server handle mismatch")
	}
	if processLease != nil {
		if handle.ProcessLease == nil || processLease.Token != handle.ProcessLease.Token {
			return fmt.Errorf("Android persistent Appium lease token mismatch")
		}
		if err := s.leaseStore.Refresh(processLease); err != nil {
			return fmt.Errorf("validate Android persistent Appium lease: %w", err)
		}
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
			using, ok := androidLocatorStrategy(strategy)
			if !ok {
				return mobile.Locator{}, true, fmt.Errorf("unsupported Android locator strategy %q", strategy)
			}
			return mobile.Locator{Using: using, Value: value}, true, nil
		case "getbytestid":
			if testIDStrategy == "accessibilityId" {
				return mobile.Locator{Using: "accessibility id", Value: value}, true, nil
			}
			return mobile.Locator{Using: "id", Value: value}, true, nil
		case "getbyaccessibilityid":
			return mobile.Locator{Using: "accessibility id", Value: value}, true, nil
		case "getbyresourceid":
			return mobile.Locator{Using: "id", Value: value}, true, nil
		case "getbyuiautomator":
			return mobile.Locator{Using: "-android uiautomator", Value: value}, true, nil
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

func androidLocatorStrategy(strategy string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "accessibilityid", "accessibility id":
		return "accessibility id", true
	case "resourceid", "id":
		return "id", true
	case "class", "classname", "class name":
		return "class name", true
	case "xpath":
		return "xpath", true
	case "uiautomator", "-android uiautomator":
		return "-android uiautomator", true
	case "css", "css selector":
		return "css selector", true
	default:
		return "", false
	}
}

func executeAndroidDevice(ctx context.Context, session *mobile.AppiumSession, call mobile.Call) (interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "back":
		return nil, session.Back(ctx)
	case "home":
		return session.Execute(ctx, "mobile: pressKey", map[string]interface{}{"keycode": 3})
	case "hidekeyboard":
		return nil, session.HideKeyboard(ctx)
	case "presskey":
		keycode, err := intCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: pressKey", map[string]interface{}{"keycode": keycode})
	case "opennotifications":
		return session.Execute(ctx, "mobile: openNotifications")
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
	case "grantpermission", "revokepermission":
		appPackage, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		permission, err := stringCallArg(call, 1)
		if err != nil {
			return nil, err
		}
		action := "grant"
		if strings.EqualFold(call.Name, "revokePermission") {
			action = "revoke"
		}
		return session.Execute(ctx, "mobile: changePermissions", map[string]interface{}{
			"appPackage": appPackage, "permissions": []string{permission}, "action": action,
		})
	case "activateapp", "terminateapp":
		appPackage, err := stringCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		script := "mobile: activateApp"
		if strings.EqualFold(call.Name, "terminateApp") {
			script = "mobile: terminateApp"
		}
		return session.Execute(ctx, script, map[string]interface{}{"appId": appPackage})
	case "backgroundapp":
		seconds, err := intCallArg(call, 0)
		if err != nil {
			return nil, err
		}
		return session.Execute(ctx, "mobile: backgroundApp", map[string]interface{}{"seconds": seconds})
	case "acceptalert":
		return nil, session.AcceptAlert(ctx)
	case "dismissalert":
		return nil, session.DismissAlert(ctx)
	case "alerttext":
		return session.AlertText(ctx)
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

func isProtectedAndroidCapability(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "platformname", "appium:automationname", "appium:udid", "appium:apppackage", "appium:appactivity", "appium:app", "udid", "automationname", "apppackage", "appactivity", "app":
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
	resourceKey := "android:avd:" + request.AVD
	if request.Serial != "" {
		resourceKey = "android:device:" + request.Serial
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
	if request.Serial != "" {
		devices, err := s.adbDevices(ctx.Background(), adb)
		if err != nil {
			return nil, err
		}
		for _, device := range devices {
			if device.Serial == request.Serial && device.State == "device" {
				lease := DeviceLease{ID: uuid.NewString(), Fence: processLease.Fence, Serial: request.Serial, AndroidSDKRoot: request.AndroidSDKRoot, ProcessLease: processLease}
				s.storeLease(lease)
				leaseCommitted = true
				s.cleanupStack(ctx).Push("device:"+lease.ID, func(cleanupCtx context.Context) error {
					_, err := s.stopOwned(cleanupCtx, adb, lease)
					return err
				})
				return &DeviceStartResponse{Lease: lease}, nil
			}
		}
		return nil, fmt.Errorf("Android device %q is not connected and ready", request.Serial)
	}

	emulator, err := resolveAndroidTool("emulator", request.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	port, serialLease, err := s.reservePort(ctx.Background(), adb, request.Port)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !leaseCommitted {
			_ = s.leaseStore.Release(serialLease)
		}
	}()
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
		if process.Stop != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = process.Stop(stopCtx)
		}
		return nil, err
	}
	if request.Animations != nil && !*request.Animations {
		if err := s.setAnimations(ctx.Background(), adb, serial, false); err != nil {
			if process.Stop != nil {
				stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = process.Stop(stopCtx)
			}
			return nil, err
		}
	}
	lease := DeviceLease{ID: uuid.NewString(), Fence: processLease.Fence, Serial: serial, AVD: request.AVD, PID: process.PID, Owned: true, LogPath: request.LogPath, AndroidSDKRoot: request.AndroidSDKRoot, ProcessLease: processLease}
	lease.SerialLease = serialLease
	s.storeLease(lease)
	leaseCommitted = true
	s.cleanupStack(ctx).Push("device:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.stopOwned(cleanupCtx, adb, lease)
		return err
	})
	return &DeviceStartResponse{Lease: lease}, nil
}

func (s *service) deviceRegister(ctx *endly.Context, request *DeviceRegisterRequest) (*DeviceRegisterResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	key := "android:external:" + strings.ToLower(strings.TrimSpace(request.Provider)) + ":" + strings.TrimSpace(request.DeviceID)
	processLease, err := s.leaseStore.Acquire(ctx.Background(), key)
	if err != nil {
		return nil, err
	}
	lease := DeviceLease{
		ID: uuid.NewString(), Fence: processLease.Fence, Serial: request.DeviceID,
		External: true, Provider: request.Provider, PlatformVersion: request.PlatformVersion,
		ProcessLease: processLease,
	}
	s.storeLease(lease)
	s.cleanupStack(ctx).Push("external-device:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.releaseExternalDevice(cleanupCtx, lease)
		return err
	})
	return &DeviceRegisterResponse{Lease: lease}, nil
}

func (s *service) deviceRelease(ctx *endly.Context, request *DeviceReleaseRequest) (*DeviceReleaseResponse, error) {
	return s.releaseExternalDevice(ctx.Background(), request.Lease)
}

func (s *service) releaseExternalDevice(_ context.Context, lease DeviceLease) (*DeviceReleaseResponse, error) {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	s.mu.Unlock()
	if !ok {
		return &DeviceReleaseResponse{Warning: "external-device registration already released or unknown"}, nil
	}
	if !stored.External || stored.Fence != lease.Fence || stored.Serial != lease.Serial {
		return nil, fmt.Errorf("external Android device lease fence mismatch")
	}
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return nil, fmt.Errorf("external Android device persistent lease token mismatch")
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
	if request.Lease.External {
		released, err := s.releaseExternalDevice(ctx.Background(), request.Lease)
		if err != nil {
			return nil, err
		}
		warning := "external device registration released; provider infrastructure was not stopped"
		if released.Warning != "" {
			warning += ": " + released.Warning
		}
		return &DeviceStopResponse{Warning: warning}, nil
	}
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
	s.mu.Unlock()
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return nil, fmt.Errorf("Android persistent device lease token mismatch")
		}
		if err := s.leaseStore.Validate(stored.ProcessLease); err != nil {
			return nil, err
		}
	}
	if !stored.Owned {
		s.mu.Lock()
		delete(s.leases, lease.ID)
		s.mu.Unlock()
		if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
			return nil, err
		}
		return &DeviceStopResponse{Warning: "attached physical/external device was released but not stopped"}, nil
	}
	if stored.SerialLease != nil {
		if err := s.leaseStore.Validate(stored.SerialLease); err != nil {
			return nil, err
		}
	}
	_, err := s.runner.Run(ctx, mobile.Command{Name: adb, Args: []string{"-s", stored.Serial, "emu", "kill"}})
	if err != nil {
		return nil, fmt.Errorf("stop emulator %s: %w", stored.Serial, err)
	}
	s.mu.Lock()
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
		return nil, err
	}
	if err := s.leaseStore.Release(stored.SerialLease); err != nil {
		return nil, err
	}
	return &DeviceStopResponse{Stopped: true}, nil
}

func (s *service) reservePort(ctx context.Context, adb string, requested int) (int, *mobile.LeaseHandle, error) {
	first, last := 5554, 5682
	if requested != 0 {
		first, last = requested, requested
	}
	for port := first; port <= last; port += 2 {
		lease, err := s.leaseStore.Acquire(ctx, "android:device:emulator-"+strconv.Itoa(port))
		if err != nil {
			if requested != 0 {
				return 0, nil, err
			}
			if errors.Is(err, mobile.ErrLeaseHeld) {
				continue
			}
			return 0, nil, err
		}
		if _, err := s.selectPort(ctx, adb, port); err != nil {
			_ = s.leaseStore.Release(lease)
			if requested != 0 {
				return 0, nil, err
			}
			continue
		}
		return port, lease, nil
	}
	return 0, nil, fmt.Errorf("no free Android emulator console port")
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
