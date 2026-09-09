package android

import (
	"fmt"
	"os"
	"strings"

	"github.com/viant/assertly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

type DoctorRequest struct {
	AndroidSDKRoot string
	Required       []string
}

func (r *DoctorRequest) Init() error {
	if r.AndroidSDKRoot == "" {
		r.AndroidSDKRoot = os.Getenv("ANDROID_SDK_ROOT")
	}
	if r.AndroidSDKRoot == "" {
		r.AndroidSDKRoot = os.Getenv("ANDROID_HOME")
	}
	if len(r.Required) == 0 {
		r.Required = []string{"adb", "emulator"}
	}
	return nil
}

type AndroidDevice struct {
	Serial string
	State  string
	Detail string
}

type DoctorResponse struct {
	Ready          bool
	AndroidSDKRoot string
	Checks         []mobile.Check
	Devices        []AndroidDevice
	AVDs           []string
}

type DeviceLease struct {
	ID             string
	Fence          uint64
	Serial         string
	AVD            string
	PID            int
	Owned          bool
	LogPath        string
	AndroidSDKRoot string
	ProcessLease   *mobile.LeaseHandle
}

type DeviceStartRequest struct {
	AndroidSDKRoot string
	AVD            string
	Serial         string
	Port           int
	WipeData       bool
	NoWindow       bool
	NoSnapshot     bool
	Animations     *bool
	BootTimeoutMs  int
	PollIntervalMs int
	LogPath        string
	EmulatorArgs   []string
}

func (r *DeviceStartRequest) Init() error {
	if r.AndroidSDKRoot == "" {
		r.AndroidSDKRoot = os.Getenv("ANDROID_SDK_ROOT")
	}
	if r.AndroidSDKRoot == "" {
		r.AndroidSDKRoot = os.Getenv("ANDROID_HOME")
	}
	if r.BootTimeoutMs <= 0 {
		r.BootTimeoutMs = 180_000
	}
	if r.PollIntervalMs <= 0 {
		r.PollIntervalMs = 1_000
	}
	if r.Animations == nil {
		disabled := false
		r.Animations = &disabled
	}
	return nil
}

func (r *DeviceStartRequest) Validate() error {
	if strings.TrimSpace(r.AVD) == "" && strings.TrimSpace(r.Serial) == "" {
		return fmt.Errorf("either AVD or Serial is required")
	}
	if r.AVD != "" && r.Serial != "" {
		return fmt.Errorf("AVD and Serial are mutually exclusive")
	}
	if r.Port != 0 && (r.Port < 5554 || r.Port > 5682 || r.Port%2 != 0) {
		return fmt.Errorf("Port must be an even emulator console port between 5554 and 5682")
	}
	return nil
}

type DeviceStartResponse struct {
	Lease DeviceLease
}

type DeviceStopRequest struct {
	Lease DeviceLease
}

func (r *DeviceStopRequest) Validate() error {
	if r.Lease.ID == "" || r.Lease.Fence == 0 || r.Lease.Serial == "" {
		return fmt.Errorf("Lease ID, Fence, and Serial are required")
	}
	return nil
}

type DeviceStopResponse struct {
	Stopped bool
	Warning string
}

type InstallRequest struct {
	Lease          DeviceLease
	APKPath        string
	APKPaths       []string
	APKSPath       string
	AABPath        string
	BundletoolPath string
	JavaPath       string
	Signing        *AndroidSigningProfile
	Package        string
	State          string
	GrantAll       bool
	AllowTest      bool
	AllowDowngrade bool
}

type AndroidSigningProfile struct {
	KeystorePath      string
	KeyAlias          string
	StorePasswordFile string
	KeyPasswordFile   string
}

func (r *InstallRequest) Init() error {
	if r.State == "" {
		r.State = "cleanData"
	}
	return nil
}

func (r *InstallRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	sources := 0
	if r.APKPath != "" {
		sources++
	}
	if len(r.APKPaths) > 0 {
		sources++
	}
	if r.APKSPath != "" {
		sources++
	}
	if r.AABPath != "" {
		sources++
	}
	if sources != 1 || r.Package == "" {
		return fmt.Errorf("exactly one of APKPath, APKPaths, APKSPath, or AABPath and Package are required")
	}
	if (r.APKSPath != "" || r.AABPath != "") && r.BundletoolPath == "" {
		return fmt.Errorf("BundletoolPath is required for APKS or AAB deployment")
	}
	if r.Signing != nil {
		if r.Signing.KeystorePath == "" || r.Signing.KeyAlias == "" || r.Signing.StorePasswordFile == "" {
			return fmt.Errorf("Signing requires KeystorePath, KeyAlias, and StorePasswordFile")
		}
	}
	switch r.State {
	case "freshInstall", "cleanData", "preserve", "upgrade":
	default:
		return fmt.Errorf("State must be freshInstall, cleanData, preserve, or upgrade")
	}
	return nil
}

type InstallResponse struct {
	Installed bool
	Package   string
	APKPath   string
	Artifacts []string
	State     string
}

type UninstallRequest struct {
	Lease   DeviceLease
	Package string
}

func (r *UninstallRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	if r.Package == "" {
		return fmt.Errorf("Package is required")
	}
	return nil
}

type UninstallResponse struct {
	Removed bool
	Package string
}

type LaunchRequest struct {
	Lease    DeviceLease
	Package  string
	Activity string
}

func (r *LaunchRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	if r.Package == "" {
		return fmt.Errorf("Package is required")
	}
	return nil
}

type LaunchResponse struct {
	Component string
	Output    string
}

type TerminateRequest struct {
	Lease   DeviceLease
	Package string
}

func (r *TerminateRequest) Validate() error {
	return (&LaunchRequest{Lease: r.Lease, Package: r.Package}).Validate()
}

type TerminateResponse struct{ Terminated bool }

type OpenRequest struct {
	SessionID      string
	Lease          DeviceLease
	Server         ServerHandle
	Package        string
	Activity       string
	TestIDStrategy string
	Capabilities   map[string]interface{}
	DescriptorPath string
	KeepSession    bool
}

func (r *OpenRequest) Init() error {
	if r.SessionID == "" {
		r.SessionID = "android-" + r.Lease.ID
	}
	return nil
}

func (r *OpenRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	if r.Server.Endpoint == "" || (r.Server.Ownership != "external" && r.Server.Ownership != "managed") {
		return fmt.Errorf("a managed or external Server handle with Endpoint is required")
	}
	if r.Package == "" {
		return fmt.Errorf("Package is required")
	}
	if r.KeepSession {
		if r.DescriptorPath == "" {
			return fmt.Errorf("DescriptorPath is required when KeepSession is enabled")
		}
		if r.Server.Ownership != "external" || r.Lease.Owned {
			return fmt.Errorf("KeepSession requires an external Appium server and an attached device")
		}
	}
	switch r.TestIDStrategy {
	case "accessibilityId", "resourceId", "composeResourceId":
	default:
		return fmt.Errorf("TestIDStrategy must be accessibilityId, resourceId, or composeResourceId")
	}
	return nil
}

type SessionHandle struct {
	ID               string
	BackendSessionID string
	Lease            DeviceLease
	Server           ServerHandle
}

type ServerHandle struct {
	ID           string
	Endpoint     string
	Ownership    string
	PID          int
	LogPath      string
	ProcessLease *mobile.LeaseHandle
}

type ServerStartRequest struct {
	Lease            DeviceLease
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
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
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

type OpenResponse struct {
	Session SessionHandle
}

// AttachRequest reconnects this Endly process to an Appium session created by
// another process. DescriptorPath is mutually exclusive with the explicit
// backend fields.
type AttachRequest struct {
	SessionID        string
	DescriptorPath   string
	BackendSessionID string
	ServerURL        string
	TargetID         string
	TestIDStrategy   string
	TakeOwnership    bool
}

func (r *AttachRequest) Validate() error {
	if r.DescriptorPath != "" {
		if r.BackendSessionID != "" || r.ServerURL != "" || r.TargetID != "" || r.TestIDStrategy != "" {
			return fmt.Errorf("DescriptorPath is mutually exclusive with explicit backend fields")
		}
		return nil
	}
	if r.BackendSessionID == "" || r.ServerURL == "" {
		return fmt.Errorf("DescriptorPath or BackendSessionID and ServerURL are required")
	}
	switch r.TestIDStrategy {
	case "accessibilityId", "resourceId", "composeResourceId":
	default:
		return fmt.Errorf("TestIDStrategy must be accessibilityId, resourceId, or composeResourceId")
	}
	return nil
}

type AttachResponse struct {
	Session SessionHandle
}

type RunRequest struct {
	SessionID        string
	Commands         []interface{}
	ActionTimeoutMs  int
	PollIntervalMs   int
	FailureArtifacts *mobile.FailureArtifactOptions
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
	Failures    []*mobile.FailureEvidence
}

func (r *RunResponse) Assertion() []*assertly.Validation { return r.Validations }

type REPLRequest struct {
	SessionID         string
	Attach            *AttachRequest
	Prompt            string
	ArtifactDirectory string
	ActionTimeoutMs   int
	PollIntervalMs    int
	MaxSourceBytes    int
	MaxTreeNodes      int
	FailOnError       bool
	HistoryPath       string
	MaxHistory        int
}

func (r *REPLRequest) Init() error {
	if r.ActionTimeoutMs <= 0 {
		r.ActionTimeoutMs = 10_000
	}
	if r.PollIntervalMs <= 0 {
		r.PollIntervalMs = 100
	}
	if r.MaxSourceBytes <= 0 {
		r.MaxSourceBytes = 2_000_000
	}
	if r.MaxTreeNodes <= 0 {
		r.MaxTreeNodes = 500
	}
	if r.MaxHistory <= 0 {
		r.MaxHistory = 1000
	}
	return nil
}

func (r *REPLRequest) Validate() error {
	if r.Attach != nil {
		if r.SessionID != "" {
			return fmt.Errorf("SessionID and Attach are mutually exclusive")
		}
		return r.Attach.Validate()
	}
	return nil
}

type REPLResponse struct {
	SessionID string
	Result    *mobile.REPLResult
}

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
	Lease          *DeviceLease
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
	if (r.SessionID == "") == (r.Lease == nil) || r.Directory == "" {
		return fmt.Errorf("exactly one of SessionID or Lease, plus Directory, is required")
	}
	if !r.Screenshot && !r.PageSource {
		return fmt.Errorf("at least one of Screenshot or PageSource is required")
	}
	if r.SessionID == "" && r.PageSource {
		return fmt.Errorf("PageSource requires SessionID")
	}
	return nil
}

type ArtifactResponse struct {
	Artifacts []*mobile.Evidence
}

type BuildRequest struct {
	ProjectDir string
	Module     string
	Variant    string
	Tasks      []string
	GradleArgs []string
	TimeoutMs  int
}

func (r *BuildRequest) Init() error {
	if r.Module == "" {
		r.Module = "app"
	}
	if r.Variant == "" {
		r.Variant = "debug"
	}
	if len(r.Tasks) == 0 {
		variant := strings.ToUpper(r.Variant[:1]) + r.Variant[1:]
		r.Tasks = []string{":" + strings.Trim(r.Module, ":/") + ":assemble" + variant}
	}
	if r.TimeoutMs <= 0 {
		r.TimeoutMs = 15 * 60 * 1000
	}
	return nil
}

func (r *BuildRequest) Validate() error {
	if r.ProjectDir == "" || len(r.Tasks) == 0 {
		return fmt.Errorf("ProjectDir and Tasks are required")
	}
	for _, arg := range append(append([]string{}, r.Tasks...), r.GradleArgs...) {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("Gradle argument contains NUL")
		}
	}
	return nil
}

type BuildArtifact struct {
	Kind   string
	Path   string
	SHA256 string
	Size   int64
}

type BuildResponse struct {
	Artifacts  []BuildArtifact
	Stdout     string
	Stderr     string
	DurationMs int
}

type CleanupRequest struct {
	Session *SessionHandle
	Server  *ServerHandle
	Lease   *DeviceLease
}

type CleanupResponse struct {
	Errors []mobile.CleanupError
}

type TestRequest struct {
	Lease       DeviceLease
	AppAPKPath  string
	TestAPKPath string
	TestPackage string
	Runner      string
	Class       string
	Arguments   map[string]string
	GrantAll    bool
	TimeoutMs   int
}

func (r *TestRequest) Init() error {
	if r.Runner == "" {
		r.Runner = "androidx.test.runner.AndroidJUnitRunner"
	}
	if r.TimeoutMs <= 0 {
		r.TimeoutMs = 15 * 60 * 1000
	}
	return nil
}

func (r *TestRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	if r.TestPackage == "" || r.Runner == "" {
		return fmt.Errorf("TestPackage and Runner are required")
	}
	if (r.AppAPKPath == "") != (r.TestAPKPath == "") {
		return fmt.Errorf("AppAPKPath and TestAPKPath must be supplied together")
	}
	return nil
}

type InstrumentationCase struct {
	Class  string
	Name   string
	Status string
	Stack  string
}

type TestResponse struct {
	Cases       []InstrumentationCase
	Passed      int
	Failed      int
	Skipped     int
	DurationMs  int
	Output      string
	Validations []*assertly.Validation
}

func (r *TestResponse) Assertion() []*assertly.Validation { return r.Validations }

type CaptureHandle struct {
	ID      string
	Lease   DeviceLease
	PID     int
	LogPath string
}

type CaptureStartRequest struct {
	Lease          DeviceLease
	Package        string
	LogPath        string
	Clear          bool
	Video          bool
	VideoDirectory string
	SegmentMs      int
}

func (r *CaptureStartRequest) Init() error {
	if r.SegmentMs <= 0 || r.SegmentMs > 170_000 {
		r.SegmentMs = 170_000
	}
	return nil
}

func (r *CaptureStartRequest) Validate() error {
	if err := (&DeviceStopRequest{Lease: r.Lease}).Validate(); err != nil {
		return err
	}
	if r.LogPath == "" && !r.Video {
		return fmt.Errorf("LogPath or Video is required")
	}
	if r.Video && r.VideoDirectory == "" {
		return fmt.Errorf("VideoDirectory is required when Video is enabled")
	}
	return nil
}

type CaptureStartResponse struct{ Capture CaptureHandle }

type CaptureStopRequest struct{ Capture CaptureHandle }

func (r *CaptureStopRequest) Validate() error {
	if r.Capture.ID == "" || r.Capture.Lease.ID == "" {
		return fmt.Errorf("Capture ID and Lease are required")
	}
	return nil
}

type CaptureStopResponse struct {
	Stopped   bool
	Artifact  *mobile.Evidence
	Artifacts []*mobile.Evidence
	Errors    []string
	Warning   string
}
