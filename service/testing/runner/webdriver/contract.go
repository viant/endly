package webdriver

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"net/url"
	"strings"
	"time"
)

const defaultTarget = "/opt/local/webdriver"
const defaultExitWaitTimeMs = 1000

type PathKind int

const (
	PathKindUndefined = PathKind(iota)
	PathKindSimple
	PathKindComposite
)

// StartRequest represents a selenium server start request
type StartRequest struct {
	Target       *location.Resource
	URL          string
	BaseLocation string
	Driver       string
	Server       string
	Sdk          string
	Capabilities []string
	// PageLoadStrategy controls how long WebDriver waits for navigation. Use
	// eager for development servers that intentionally keep a live connection.
	PageLoadStrategy string
	Port             int
}

func (r *StartRequest) Init() error {
	if r.Port == 0 {
		r.Port = 4444
	}
	if r.Driver == "" {
		r.Driver = ChromeDriver
	}
	if r.Target == nil && r.URL != "" {
		r.Target = location.NewResource(r.URL)
	}
	if r.Target == nil {
		r.Target = location.NewResource(defaultTarget)
	}
	return nil
}

func (r *StartRequest) Validate() error {
	return nil
}

// NewStartRequestFromURL creates a new start request from URL
func NewStartRequestFromURL(URL string) (*StartRequest, error) {
	var result = &StartRequest{}
	var resource = location.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

// StartResponse represents a selenium server stop request
type StartResponse struct {
	Pid        int
	ServerPath string
	DriverPath string
	SessionID  string
}

// StopRequest represents server stop request
type StopRequest struct {
	Target *location.Resource
	URL    string
	Port   int
}

func (r *StopRequest) Init() error {
	if r.Port == 0 {
		r.Port = 4444
	}
	if r.Target == nil && r.URL != "" {
		r.Target = location.NewResource(r.URL)
	}
	if r.Target == nil {
		r.Target = location.NewResource(defaultTarget)
	}
	return nil

}

// NewStopRequestFromURL creates a new start request from URL
func NewStopRequestFromURL(URL string) (*StopRequest, error) {
	var result = &StopRequest{}
	var resource = location.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

// StopResponse represents a selenium stop request
type StopResponse struct {
}

// OpenSessionResponse represents open session response.
type OpenSessionResponse struct {
	SessionID string
	Attached  bool
	Backend   string
}

// CloseSessionRequest represents close session request.
type CloseSessionRequest struct {
	SessionID string
}

// NewCloseSessionRequestFromURL creates a new close session request from URL
func NewCloseSessionRequestFromURL(URL string) (*CloseSessionRequest, error) {
	var result = &CloseSessionRequest{}
	var resource = location.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

// CloseSessionResponse represents close session response.
type CloseSessionResponse struct {
	SessionID string
}

type StopLoadingRequest struct {
	SessionID string
}

type StopLoadingResponse struct {
	SessionID string
	Stopped   bool
}

// WebDriverCallRequest represents selenium call driver request
type WebDriverCallRequest struct {
	SessionID     string
	Key           string
	PathKind      PathKind
	Call          *MethodCall
	sessionLocked bool
}

// ServiceCallResponse represents selenium call response
type ServiceCallResponse struct {
	Result []interface{}
	Data   data.Map
}

// WebElementSelector represents a web element selector
type WebElementSelector struct {
	By    string //selector type
	Value string //selector value
	Key   string //optional result key
}

// WebElementCallRequest represents a web element call reqesut
type WebElementCallRequest struct {
	SessionID            string
	Selector             *WebElementSelector
	Call                 *MethodCall
	PathKind             PathKind
	AllowJavaScriptClick bool
	Strict               bool
	sessionLocked        bool
}

// WebElementCallResponse represents seleniun web element response
type WebElementCallResponse struct {
	Result      []interface{}
	LookupError string
	Data        map[string]interface{}
}

// RunRequest represents group of selenium web elements calls
type RunRequest struct {
	SessionID            string
	Browser              string
	RemoteSelenium       string             //remote selenium resource
	DebuggerAddress      string             `description:"optional host:port of an existing Chrome started with remote debugging enabled"`
	DirectCDP            bool               `description:"connect directly to a debug-enabled Chrome without ChromeDriver"`
	PageLoadStrategy     string             `description:"browser navigation strategy: normal, eager, or none"`
	BlockedURLs          []string           `description:"optional Chrome URL patterns blocked through CDP to reduce test noise and load"`
	AutoStart            *bool              `description:"automatically start a local driver for the default session; defaults to true"`
	Navigation           *NavigationOptions `description:"optional Get(url) navigation guard options"`
	Actions              []*Action
	ActionTimeoutMs      int                     `description:"default total deadline for each locator action; defaults to 10000ms"`
	PollIntervalMs       int                     `description:"default polling interval for locator actions; defaults to 100ms"`
	AllowJavaScriptClick bool                    `description:"opt-in fallback to DOM click after a native click is intercepted or not interactable"`
	StrictSelectors      *bool                   `description:"override locator strictness; Playwright-style locators are strict by default"`
	FailureArtifacts     *FailureArtifactOptions `description:"optional screenshot, page-source, and browser-capture evidence written when the run fails"`
	ActionDelaysMs       int                     `description:"slows down action with specified delay"`
	Commands             []interface{}           `description:"list of selenium command: {web element selector}.WebElementMethod(params),  or WebDriverMethod(params), or wait map "`
	Expect               interface{}             `description:"If specified it will validated response as actual"`
}

type NavigationOptions struct {
	TimeoutMs            int    `description:"page load timeout for Get(url); on timeout it warns and continues"`
	ContinueOnTimeout    *bool  `description:"continue after a page-load timeout; defaults to true"`
	StopLoadingOnTimeout *bool  `description:"call window.stop after a page-load timeout; defaults to true"`
	AutoScrollMs         int    `description:"hard time budget for bounded lazy-content scrolling; zero disables scrolling"`
	ScrollSelector       string `description:"optional CSS selector for a scroll container; defaults to the document"`
	ScrollDelayMs        int    `description:"delay between scroll steps"`
	StableWindowMs       int    `description:"stop autoscroll after height and position remain stable at the bottom"`
	MaxScrollSteps       int    `description:"hard limit on scroll steps"`
	MaxScrollGrowthPx    int    `description:"hard limit on document-height growth; protects against infinite feeds"`
	MaxScrollHeightPx    int    `description:"optional absolute document-height limit"`
	ReturnToTop          bool   `description:"return to the top after lazy-content scrolling"`
	IdleThreshold        int    `description:"network idle threshold (inflight requests <= threshold)"`
	IdleWindowMs         int    `description:"consider network idle only if threshold holds for this long"`
	IdleMaxWaitMs        int    `description:"optional smaller network-idle budget; never extends AutoScrollMs"`
}

// NavigationReport explains how navigation stabilization stopped. It makes
// infinite-feed and slow-page behavior observable instead of silently hiding it.
type NavigationReport struct {
	URL            string
	TimedOut       bool
	Scrolled       bool
	Steps          int
	StartHeightPx  int
	FinalHeightPx  int
	ElapsedMs      int
	StopReason     string
	ScrollTarget   string
	LoadingStopped bool
	Warning        string
}

type FailureArtifactOptions struct {
	Directory      string `description:"AFS URL or local directory for failure evidence; required when enabled"`
	Screenshot     *bool  `description:"capture a PNG screenshot; defaults to true"`
	PageSource     *bool  `description:"capture current page HTML; defaults to true"`
	IncludeCapture bool   `description:"include buffered console and network capture in metadata"`
	MaxSourceBytes int    `description:"maximum page-source bytes; defaults to 2000000"`
}

type FailureArtifact struct {
	Timestamp     time.Time
	Error         string
	Method        string
	Selector      string
	URL           string
	Title         string
	ScreenshotURL string
	PageSourceURL string
	MetadataURL   string
	Capture       *CaptureSummary
	CaptureErrors []string
}

type AssertionResult struct {
	Method    string
	Selector  string
	Matcher   string
	Expected  interface{}
	Actual    interface{}
	Passed    bool
	Error     string
	ElapsedMs int
}

func (r *RunRequest) asWaitAction(parser *parser, candidate interface{}) (*Action, error) {
	if aMap, err := util.NormalizeMap(candidate, true); err == nil {
		command, ok := aMap["command"]
		if !ok {
			return nil, fmt.Errorf("command was missing: %v", candidate)
		}
		action, err := parser.Parse(toolbox.AsString(command))
		if err != nil {
			return nil, err
		}
		if action.PathKind == PathKindUndefined {
			action.PathKind = PathKindSimple
		}
		applyWaitOverrides(&action.Calls[0].Wait, aMap)
		call := action.Calls[0]
		_, hasExplicitWait := aMap["waitTimeMs"]
		repeat := toolbox.AsInt(aMap["repeat"])
		if !hasExplicitWait && repeat > 0 {
			sleepTimeMs := toolbox.AsInt(aMap["sleepTimeMs"])
			if sleepTimeMs <= 0 {
				sleepTimeMs = defaultExitWaitTimeMs
			}
			call.WaitTimeMs = repeat * sleepTimeMs
		} else if call.WaitTimeMs == 0 && strings.TrimSpace(call.Exit) != "" {
			call.WaitTimeMs = defaultExitWaitTimeMs
		}
		return action, err
	}
	return nil, fmt.Errorf("sunupported command: %T", candidate)
}

func applyWaitOverrides(wait *Wait, values map[string]interface{}) {
	if wait == nil {
		return
	}
	for key, value := range values {
		switch strings.ToLower(key) {
		case "waittimems":
			wait.WaitTimeMs = toolbox.AsInt(value)
		case "pollintervalms":
			wait.PollIntervalMs = toolbox.AsInt(value)
		case "thinktimems":
			wait.ThinkTimeMs = toolbox.AsInt(value)
		case "ignoretimeout":
			wait.IgnoreTimeout = toolbox.AsBoolean(value)
		case "exit":
			wait.Exit = toolbox.AsString(value)
		}
	}
}

func (r *RunRequest) Init() error {
	if r.ActionTimeoutMs <= 0 {
		r.ActionTimeoutMs = 10_000
	}
	if r.PollIntervalMs <= 0 {
		r.PollIntervalMs = 100
	}
	switch strings.ToLower(r.PageLoadStrategy) {
	case "", "normal", "eager", "none":
	default:
		return fmt.Errorf("invalid pageLoadStrategy %q", r.PageLoadStrategy)
	}
	if r.DebuggerAddress != "" && r.Browser == "" {
		r.Browser = ChromeBrowser
	}
	if r.SessionID == "" && r.RemoteSelenium != "" {
		if parsed, err := url.Parse(r.RemoteSelenium); err == nil && parsed.Host != "" {
			r.SessionID = parsed.Host
		}
	}
	if r.SessionID == "" {
		r.SessionID = "localhost:4444"
	}
	if len(r.Actions) > 0 {
		for _, action := range r.Actions {
			if action.Selector != nil {
				_ = action.Selector.Init()
				if action.Selector.Key == "" {
					action.Selector.Key = action.Key
				}
				if action.Selector.Key == "" {
					action.Selector.Key = action.Selector.Value
				}
			}
		}
		return nil
	}
	if len(r.Commands) == 0 {
		return nil
	}

	expectMap := r.expectMap()
	r.Actions = make([]*Action, 0)
	var previousAction *Action
	parser := &parser{}
	for _, candidate := range r.Commands {
		command, ok := candidate.(string)
		if !ok {
			action, err := r.asWaitAction(parser, candidate)
			if err != nil {
				return err
			}
			r.setWaitExitIfNeeded(action.Calls[0], expectMap, action)
			r.Actions = append(r.Actions, action)
			continue
		}
		action, err := parser.Parse(command)
		if err != nil {
			return fmt.Errorf("invalid command: %v, %v", command, err)
		}
		if previousAction != nil {
			if previousAction.Selector != nil && action.Selector != nil && previousAction.Selector.Value == action.Selector.Value {
				if action.Key == "" && isReadMethod(action.Calls[0].Method) && isReadMethod(previousAction.Calls[0].Method) {
					previousAction.Calls = append(previousAction.Calls, action.Calls[0])
					previousAction.PathKind = PathKindComposite
					continue
				}
			}
		}
		r.Actions = append(r.Actions, action)
		previousAction = action
		call := action.Calls[0]
		r.setWaitExitIfNeeded(call, expectMap, action)

	}
	return nil
}

func (r *RunRequest) setWaitExitIfNeeded(call *MethodCall, expectMap map[string]interface{}, action *Action) {
	if call.Exit == "" {
		if expectValue, ok := expectMap[action.Key]; ok && expectValue != nil {
			switch actual := expectValue.(type) {
			case string:
				call.Exit = "$" + action.Key + " contains " + actual
			case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
				call.Exit = "$" + action.Key + " = " + toolbox.AsString(expectValue)
			case float64:
				call.Exit = "$" + action.Key + " = " + toolbox.AsString(expectValue)
			}
			if call.Exit != "" {
				if call.WaitTimeMs == 0 {
					call.WaitTimeMs = defaultExitWaitTimeMs
				}
				call.IgnoreTimeout = true
			}
		}
	}
}

func isReadMethod(method string) bool {
	return strings.HasPrefix(method, "Get") || strings.HasPrefix(method, "Text")
}

// NewRunRequest creates a new run request
func NewRunRequest(sessionID, browser string, remote string, actions ...*Action) *RunRequest {
	return &RunRequest{
		SessionID:      sessionID,
		Browser:        browser,
		RemoteSelenium: remote,
		Actions:        actions,
	}
}

// NewRunRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	resource := location.NewResource(URL)
	var result = &RunRequest{}
	return result, resource.Decode(result)
}

// RunResponse represents selenium call response
type RunResponse struct {
	SessionID    string
	Backend      string
	Data         map[string]interface{}
	LookupErrors []string
	Navigations  []*NavigationReport
	Failures     []*FailureArtifact
	Assertions   []*AssertionResult
	Assert       *validator.AssertResponse
}

// Assertion exposes webdriver validation to the CLI/xUnit reporting pipeline.
func (r *RunResponse) Assertion() []*assertly.Validation {
	result := make([]*assertly.Validation, 0, 2)
	if r == nil {
		return result
	}
	if len(r.Assertions) > 0 {
		validation := assertly.NewValidation()
		validation.Description = "webdriver inline expectations"
		for _, assertion := range r.Assertions {
			if assertion == nil {
				continue
			}
			if assertion.Passed {
				validation.PassedCount++
				continue
			}
			reason := assertly.EqualViolation
			if assertion.Matcher == "contains" {
				reason = assertly.ContainsViolation
			}
			validation.AddFailure(assertly.NewFailure("webdriver", assertion.Selector, reason, assertion.Expected, assertion.Actual))
		}
		result = append(result, validation)
	}
	if r.Assert != nil {
		result = append(result, r.Assert.Assertion()...)
	}
	return result
}

type CaptureStartRequest struct {
	SessionID       string
	SinkURL         string `description:"optional AFS URL for JSONL event sink (file://...)"` // proposal C
	FlushIntervalMs int    `description:"optional sink sync interval in ms (file sinks only)"`
	MaxBodyBytes    int
	MaxEntries      int `description:"maximum retained console and completed-network entries; defaults to 10000"`
	Redact          *bool
	RedactHeaders   []string
	EnableConsole   *bool
	EnableNetwork   *bool
	IncludeBodies   *bool
	URLIncludes     []string `description:"optional URL substrings; when set, capture only matching network requests"`
}

type CaptureStartResponse struct {
	SessionID string
	Enabled   bool
	Warning   string
}

type CaptureStopRequest struct {
	SessionID string
}

type CaptureStopResponse struct {
	SessionID string
	Summary   *CaptureSummary
}

type CaptureStatusRequest struct {
	SessionID string
}

type CaptureStatusResponse struct {
	SessionID string
	Summary   *CaptureSummary
}

type CaptureClearRequest struct {
	SessionID string
}

type CaptureClearResponse struct {
	SessionID string
}

type CaptureExportRequest struct {
	SessionID      string
	MaxEntries     int
	IncludeConsole *bool
	IncludeNetwork *bool
}

type CaptureExportResponse struct {
	SessionID string
	Summary   *CaptureSummary
	Console   []*ConsoleEntry
	Network   []*NetworkTransaction
}

// MethodCall represents selenium call.
type MethodCall struct {
	Wait
	Method       string
	Parameters   []interface{}
	AllowMissing bool `yaml:"-" json:"-"`
}

type Wait struct {
	WaitTimeMs     int
	PollIntervalMs int `description:"poll interval for wait conditions; defaults to 100ms"`
	ThinkTimeMs    int
	IgnoreTimeout  bool
	Exit           string
	Expectation    *CallExpectation `yaml:"-" json:"-"`
	criteria       eval.Compute
}

type CallExpectation struct {
	Matcher string
	Value   interface{}
}

// Action represents various calls on web element
type Action struct {
	Key string //optional result key
	PathKind
	Selector *WebElementSelector
	Calls    []*MethodCall
	Strict   bool
}

// NewAction creates a new action
func NewAction(key, selector string, method string, params ...interface{}) *Action {
	var result = &Action{
		Key:      key,
		PathKind: PathKindSimple,
		Calls: []*MethodCall{
			{
				Method:     method,
				Parameters: params,
			},
		},
	}
	if selector != "" {
		var webSelector = WebSelector(selector)
		result.Selector = &WebElementSelector{}
		result.Selector.By, result.Selector.Value = webSelector.ByAndValue()
		result.Selector.Key = result.Key
	}
	return result
}

// Validate validates run request.
func (r *RunRequest) Validate() error {
	if r.FailureArtifacts != nil && strings.TrimSpace(r.FailureArtifacts.Directory) == "" {
		return fmt.Errorf("failureArtifacts.directory was empty")
	}
	if r.SessionID == "" {
		if r.Browser == "" {
			return fmt.Errorf("both SessionID and Browser were empty")
		}
	}

	if len(r.Actions) == 0 {
		return fmt.Errorf("both actions/commands were empty")
	}

	for i, action := range r.Actions {
		if len(action.Calls) == 0 {
			return fmt.Errorf("actions[%d].Calls were empty", i)
		}
	}
	return nil
}

// OpenSessionRequest represents open session request
type OpenSessionRequest struct {
	Browser          string
	Capabilities     []string
	Remote           string   `description:"webdriver server endpoint"`
	SessionID        string   `description:"if specified this SessionID will be used for a sessionID"`
	DebuggerAddress  string   `description:"optional host:port of an existing Chrome started with remote debugging enabled"`
	DirectCDP        bool     `description:"connect directly to a debug-enabled Chrome without ChromeDriver"`
	PageLoadStrategy string   `description:"browser navigation strategy: normal, eager, or none"`
	BlockedURLs      []string `description:"optional Chrome URL patterns blocked through CDP"`
}

// Init  initializes request
func (r *OpenSessionRequest) Init() error {
	if r.DirectCDP && strings.TrimSpace(r.DebuggerAddress) == "" {
		return fmt.Errorf("debuggerAddress was required for directCDP")
	}
	if r.DebuggerAddress != "" && r.Browser == "" {
		r.Browser = ChromeBrowser
	}
	if r.SessionID == "" && r.Remote != "" {
		if parsed, err := url.Parse(r.Remote); err == nil && parsed.Host != "" {
			r.SessionID = parsed.Host
		}
	}
	if r.SessionID == "" {
		r.SessionID = "localhost:4444"
	}
	if r.Remote == "" {
		host, port := pair(r.SessionID)
		r.Remote = fmt.Sprintf("http://%v:%v/wd/hub", host, port)
	}
	switch strings.ToLower(r.PageLoadStrategy) {
	case "", "normal", "eager", "none":
	default:
		return fmt.Errorf("invalid pageLoadStrategy %q", r.PageLoadStrategy)
	}
	return nil
}

// NewOpenSessionRequest creates a new open session request
func NewOpenSessionRequest(browser string, remote string) *OpenSessionRequest {
	return &OpenSessionRequest{
		Browser: browser,
		Remote:  remote,
	}
}

func (r *RunRequest) expectMap() map[string]interface{} {
	var expectMap = map[string]interface{}{}
	if r.Expect != nil {
		if value, ok := r.Expect.(map[string]interface{}); ok {
			expectMap = value
		}
	}
	return expectMap
}
