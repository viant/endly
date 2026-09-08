package ios

import (
	"fmt"
	"strings"

	"github.com/viant/assertly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

type DoctorRequest struct {
	Required []string
}

func (r *DoctorRequest) Init() error {
	if len(r.Required) == 0 {
		r.Required = []string{"xcodebuild", "simulator-runtime"}
	}
	return nil
}

type IOSSimulator struct {
	UDID        string
	Name        string
	State       string
	Runtime     string
	IsAvailable bool
}

type DoctorResponse struct {
	Ready      bool
	Checks     []mobile.Check
	Runtimes   []string
	Simulators []IOSSimulator
}

type DestinationLease struct {
	ID         string
	Fence      uint64
	UDID       string
	Name       string
	Runtime    string
	OwnedClone bool
}

type SimulatorStartRequest struct {
	UDID          string
	BaseName      string
	CloneName     string
	DeviceType    string
	Runtime       string
	Erase         bool
	BootTimeoutMs int
}

func (r *SimulatorStartRequest) Init() error {
	if r.BootTimeoutMs <= 0 {
		r.BootTimeoutMs = 180_000
	}
	return nil
}

func (r *SimulatorStartRequest) Validate() error {
	selectors := 0
	if r.UDID != "" {
		selectors++
	}
	if r.BaseName != "" {
		selectors++
	}
	if r.DeviceType != "" || r.Runtime != "" {
		if r.DeviceType == "" || r.Runtime == "" {
			return fmt.Errorf("DeviceType and Runtime must be supplied together")
		}
		selectors++
	}
	if selectors != 1 {
		return fmt.Errorf("exactly one of UDID, BaseName, or DeviceType+Runtime is required")
	}
	if r.CloneName != "" && r.BaseName == "" {
		return fmt.Errorf("CloneName requires BaseName")
	}
	if strings.TrimSpace(r.CloneName) == strings.TrimSpace(r.BaseName) && r.CloneName != "" {
		return fmt.Errorf("CloneName must differ from BaseName")
	}
	return nil
}

type SimulatorStartResponse struct {
	Lease DestinationLease
}

type SimulatorStopRequest struct {
	Lease DestinationLease
}

func (r *SimulatorStopRequest) Validate() error {
	if r.Lease.ID == "" || r.Lease.Fence == 0 || r.Lease.UDID == "" {
		return fmt.Errorf("Lease ID, Fence, and UDID are required")
	}
	return nil
}

type SimulatorStopResponse struct {
	Shutdown bool
	Deleted  bool
	Warning  string
}

type InstallRequest struct {
	Destination DestinationLease
	App         Artifact
	BundleID    string
	State       string
}

func (r *InstallRequest) Init() error {
	if r.State == "" {
		r.State = "freshInstall"
	}
	return nil
}

func (r *InstallRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.App.HostPath == "" || r.BundleID == "" {
		return fmt.Errorf("App.HostPath and BundleID are required")
	}
	if r.App.Kind != "" && r.App.Kind != "simulatorApp" {
		return fmt.Errorf("current Simulator install requires a simulatorApp artifact")
	}
	if r.State != "freshInstall" && r.State != "preserve" {
		return fmt.Errorf("State must be freshInstall or preserve")
	}
	return nil
}

type InstallResponse struct {
	Installed bool
	BundleID  string
	AppPath   string
	State     string
}

type UninstallRequest struct {
	Destination DestinationLease
	BundleID    string
}

func (r *UninstallRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.BundleID == "" {
		return fmt.Errorf("BundleID is required")
	}
	return nil
}

type UninstallResponse struct {
	Removed  bool
	BundleID string
}

type LaunchRequest struct {
	Destination DestinationLease
	BundleID    string
	Arguments   []string
	Environment map[string]string
}

func (r *LaunchRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.BundleID == "" {
		return fmt.Errorf("BundleID is required")
	}
	return nil
}

type LaunchResponse struct {
	PID    int
	Output string
}

type TerminateRequest struct {
	Destination DestinationLease
	BundleID    string
}

func (r *TerminateRequest) Validate() error {
	return (&LaunchRequest{Destination: r.Destination, BundleID: r.BundleID}).Validate()
}

type TerminateResponse struct{ Terminated bool }

type OpenRequest struct {
	SessionID    string
	Destination  DestinationLease
	Server       ServerHandle
	BundleID     string
	App          *Artifact
	Capabilities map[string]interface{}
}

func (r *OpenRequest) Init() error {
	if r.SessionID == "" {
		r.SessionID = "ios-" + r.Destination.ID
	}
	return nil
}

func (r *OpenRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.Server.Endpoint == "" || r.Server.Ownership != "external" {
		return fmt.Errorf("an external Server handle with Endpoint is required")
	}
	if r.BundleID == "" && r.App == nil {
		return fmt.Errorf("BundleID or App is required")
	}
	return nil
}

type SessionHandle struct {
	ID               string
	BackendSessionID string
	Destination      DestinationLease
	Server           ServerHandle
}

type ServerHandle struct {
	ID        string
	Endpoint  string
	Ownership string
	PID       int
	LogPath   string
}

type ServerStartRequest struct {
	Destination      DestinationLease
	Mode             string
	ServerURL        string
	Executable       string
	AppiumHome       string
	Address          string
	Port             int
	BasePath         string
	LogPath          string
	StartupTimeoutMs int
}

func (r *ServerStartRequest) Init() error {
	options := r.options()
	options.Init()
	r.Mode, r.Executable, r.Address, r.Port = options.Mode, options.Executable, options.Address, options.Port
	r.BasePath, r.StartupTimeoutMs = options.BasePath, options.StartupTimeoutMs
	return nil
}

func (r *ServerStartRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	return r.options().Validate()
}

func (r *ServerStartRequest) options() mobile.AppiumServerOptions {
	return mobile.AppiumServerOptions{
		Mode: r.Mode, ServerURL: r.ServerURL, Executable: r.Executable,
		AppiumHome: r.AppiumHome, Address: r.Address, Port: r.Port,
		BasePath: r.BasePath, LogPath: r.LogPath, StartupTimeoutMs: r.StartupTimeoutMs,
	}
}

type ServerStartResponse struct{ Server ServerHandle }

type ServerStopRequest struct{ Server ServerHandle }

func (r *ServerStopRequest) Validate() error {
	if r.Server.ID == "" || r.Server.Endpoint == "" {
		return fmt.Errorf("Server ID and Endpoint are required")
	}
	return nil
}

type ServerStopResponse struct {
	Stopped bool
	Warning string
}

type Artifact struct {
	Kind        string
	BundleID    string
	HostPath    string
	ArtifactURL string
	SHA256      string
	Size        int64
}

type OpenResponse struct {
	Session SessionHandle
}

type RunRequest struct {
	SessionID       string
	Commands        []string
	ActionTimeoutMs int
	PollIntervalMs  int
}

func (r *RunRequest) Init() error {
	if r.ActionTimeoutMs <= 0 {
		r.ActionTimeoutMs = 10_000
	}
	if r.PollIntervalMs <= 0 {
		r.PollIntervalMs = 100
	}
	return nil
}

func (r *RunRequest) Validate() error {
	if r.SessionID == "" {
		return fmt.Errorf("SessionID is required")
	}
	if len(r.Commands) == 0 {
		return fmt.Errorf("Commands are required")
	}
	return nil
}

type RunResponse struct {
	Data        map[string]interface{}
	Steps       []mobile.ExecutionStep
	Validations []*assertly.Validation
}

func (r *RunResponse) Assertion() []*assertly.Validation { return r.Validations }

type CloseRequest struct {
	SessionID string
}

func (r *CloseRequest) Validate() error {
	if r.SessionID == "" {
		return fmt.Errorf("SessionID is required")
	}
	return nil
}

type CloseResponse struct {
	Closed  bool
	Warning string
}

type ArtifactRequest struct {
	SessionID      string
	Destination    *DestinationLease
	Directory      string
	Screenshot     bool
	PageSource     bool
	MaxSourceBytes int
}

func (r *ArtifactRequest) Init() error {
	if r.MaxSourceBytes <= 0 {
		r.MaxSourceBytes = 2_000_000
	}
	return nil
}

func (r *ArtifactRequest) Validate() error {
	if (r.SessionID == "") == (r.Destination == nil) || r.Directory == "" {
		return fmt.Errorf("exactly one of SessionID or Destination, plus Directory, is required")
	}
	if r.SessionID == "" && r.PageSource {
		return fmt.Errorf("PageSource requires SessionID")
	}
	if !r.Screenshot && !r.PageSource {
		return fmt.Errorf("at least one of Screenshot or PageSource is required")
	}
	return nil
}

type ArtifactResponse struct {
	Artifacts []*mobile.Evidence
}

type BuildRequest struct {
	ProjectPath     string
	WorkspacePath   string
	Scheme          string
	Configuration   string
	Destination     DestinationLease
	DerivedDataPath string
	Mode            string
	BuildSettings   map[string]string
	TimeoutMs       int
}

func (r *BuildRequest) Init() error {
	if r.Configuration == "" {
		r.Configuration = "Debug"
	}
	if r.Mode == "" {
		r.Mode = "build"
	}
	if r.TimeoutMs <= 0 {
		r.TimeoutMs = 20 * 60 * 1000
	}
	return nil
}

func (r *BuildRequest) Validate() error {
	if (r.ProjectPath == "") == (r.WorkspacePath == "") {
		return fmt.Errorf("exactly one of ProjectPath or WorkspacePath is required")
	}
	if r.Scheme == "" || r.DerivedDataPath == "" {
		return fmt.Errorf("Scheme and DerivedDataPath are required")
	}
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.Mode != "build" && r.Mode != "buildForTesting" {
		return fmt.Errorf("Mode must be build or buildForTesting")
	}
	for key, value := range r.BuildSettings {
		if strings.ContainsRune(key+value, '\x00') || strings.ContainsAny(key, " \t=") {
			return fmt.Errorf("invalid Xcode build setting %q", key)
		}
	}
	return nil
}

type BuildResponse struct {
	Artifacts  []Artifact
	Stdout     string
	Stderr     string
	DurationMs int
}

type CleanupRequest struct {
	Session     *SessionHandle
	Server      *ServerHandle
	Destination *DestinationLease
}

type CleanupResponse struct {
	Errors []mobile.CleanupError
}

type TestRequest struct {
	Destination      DestinationLease
	ProjectPath      string
	WorkspacePath    string
	Scheme           string
	XCTestRunPath    string
	Mode             string
	OnlyTesting      []string
	SkipTesting      []string
	ResultBundlePath string
	TimeoutMs        int
}

func (r *TestRequest) Init() error {
	if r.Mode == "" {
		r.Mode = "scheme"
	}
	if r.TimeoutMs <= 0 {
		r.TimeoutMs = 20 * 60 * 1000
	}
	return nil
}

func (r *TestRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.ResultBundlePath == "" {
		return fmt.Errorf("ResultBundlePath is required")
	}
	switch r.Mode {
	case "scheme", "withoutBuilding":
		if (r.ProjectPath == "") == (r.WorkspacePath == "") || r.Scheme == "" {
			return fmt.Errorf("scheme modes require exactly one project/workspace path and Scheme")
		}
	case "xctestrun":
		if r.XCTestRunPath == "" {
			return fmt.Errorf("xctestrun mode requires XCTestRunPath")
		}
	default:
		return fmt.Errorf("Mode must be scheme, withoutBuilding, or xctestrun")
	}
	return nil
}

type XCTestFailure struct {
	TestName       string `json:"testName"`
	TargetName     string `json:"targetName"`
	FailureText    string `json:"failureText"`
	TestIdentifier string `json:"testIdentifierString"`
}

type XCTestSummary struct {
	Title            string          `json:"title"`
	Result           string          `json:"result"`
	TotalTestCount   int             `json:"totalTestCount"`
	PassedTests      int             `json:"passedTests"`
	FailedTests      int             `json:"failedTests"`
	SkippedTests     int             `json:"skippedTests"`
	ExpectedFailures int             `json:"expectedFailures"`
	TestFailures     []XCTestFailure `json:"testFailures"`
}

type TestResponse struct {
	Summary     XCTestSummary
	Output      string
	DurationMs  int
	Validations []*assertly.Validation
}

func (r *TestResponse) Assertion() []*assertly.Validation { return r.Validations }

type CaptureHandle struct {
	ID          string
	Destination DestinationLease
	PID         int
	LogPath     string
}

type CaptureStartRequest struct {
	Destination DestinationLease
	Predicate   string
	LogPath     string
}

func (r *CaptureStartRequest) Validate() error {
	if err := (&SimulatorStopRequest{Lease: r.Destination}).Validate(); err != nil {
		return err
	}
	if r.LogPath == "" {
		return fmt.Errorf("LogPath is required")
	}
	return nil
}

type CaptureStartResponse struct{ Capture CaptureHandle }

type CaptureStopRequest struct{ Capture CaptureHandle }

func (r *CaptureStopRequest) Validate() error {
	if r.Capture.ID == "" || r.Capture.Destination.ID == "" {
		return fmt.Errorf("Capture ID and Destination are required")
	}
	return nil
}

type CaptureStopResponse struct {
	Stopped  bool
	Artifact *mobile.Evidence
	Warning  string
}
