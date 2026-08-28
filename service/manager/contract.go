package manager

import (
	"time"

	"github.com/viant/endly/internal/debug"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
)

const ServiceID = "manager"

const (
	OperationQueued     = "queued"
	OperationRunning    = "running"
	OperationSucceeded  = "succeeded"
	OperationFailed     = "failed"
	OperationCancelling = "cancelling"
	OperationCancelled  = "cancelled"
)

type OpenRequest struct {
	Name    string `json:"name,omitempty"`
	Subject string `json:"-"`
}

type SessionInfo struct {
	SessionID string    `json:"sessionId"`
	Name      string    `json:"name,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListSessionsResponse struct {
	Sessions []*SessionInfo `json:"sessions"`
}

type CloseRequest struct {
	SessionID string `json:"sessionId"`
}

type LoadWorkflowRequest struct {
	SessionID string `json:"sessionId"`
	URL       string `json:"url"`
	Name      string `json:"name,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Replace   bool   `json:"replace,omitempty"`
	Content   string `json:"content,omitempty"`
	Format    string `json:"format,omitempty"`
}

type LoadedWorkflow struct {
	SessionID string   `json:"sessionId"`
	Alias     string   `json:"alias"`
	Name      string   `json:"name"`
	Source    string   `json:"source"`
	Tasks     []string `json:"tasks"`
}

type ListWorkflowsRequest struct {
	SessionID string `json:"sessionId"`
}

type ListWorkflowsResponse struct {
	SessionID string            `json:"sessionId"`
	Workflows []*LoadedWorkflow `json:"workflows"`
}

type UnloadWorkflowRequest struct {
	SessionID string `json:"sessionId"`
	Workflow  string `json:"workflow"`
	Force     bool   `json:"force,omitempty"`
}

type RunWorkflowRequest struct {
	SessionID         string                 `json:"sessionId"`
	Workflow          string                 `json:"workflow"`
	Tasks             string                 `json:"tasks,omitempty"`
	TagIDs            string                 `json:"tagIds,omitempty"`
	Params            map[string]interface{} `json:"params,omitempty"`
	SharedState       *bool                  `json:"sharedState,omitempty"`
	Callbacks         []Callback             `json:"callbacks,omitempty"`
	Debug             bool                   `json:"debug,omitempty"`
	Breakpoints       []debug.Step           `json:"breakpoints,omitempty"`
	Timeout           time.Duration          `json:"-"`
	TimeoutMillis     int64                  `json:"timeoutMillis,omitempty"`
	PublishParameters *bool                  `json:"publishParameters,omitempty"`
}

type RunActionRequest struct {
	SessionID     string                 `json:"sessionId"`
	Service       string                 `json:"service"`
	Action        string                 `json:"action"`
	Request       map[string]interface{} `json:"request,omitempty"`
	Callbacks     []Callback             `json:"callbacks,omitempty"`
	TimeoutMillis int64                  `json:"timeoutMillis,omitempty"`
}

type StartOperationRequest struct {
	Kind     string              `json:"kind"`
	Workflow *RunWorkflowRequest `json:"workflow,omitempty"`
	Action   *RunActionRequest   `json:"action,omitempty"`
}

type Callback struct {
	URL     string            `json:"url"`
	Events  []string          `json:"events,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Operation struct {
	ID            string                 `json:"id"`
	SessionID     string                 `json:"sessionId"`
	Status        string                 `json:"status"`
	Kind          string                 `json:"kind"`
	Workflow      string                 `json:"workflow"`
	Service       string                 `json:"service,omitempty"`
	Action        string                 `json:"action,omitempty"`
	Tasks         string                 `json:"tasks"`
	TagIDs        string                 `json:"tagIds,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	StartedAt     *time.Time             `json:"startedAt,omitempty"`
	FinishedAt    *time.Time             `json:"finishedAt,omitempty"`
	Result        map[string]interface{} `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Events        []*OperationEvent      `json:"events,omitempty"`
	Debug         *DebugState            `json:"debug,omitempty"`
	DroppedEvents int64                  `json:"droppedEvents,omitempty"`

	cancel        func()
	debugger      *debug.Debugger
	eventQueue    chan *queuedEvent
	eventDone     chan struct{}
	droppedEvents int64
}

type OperationEvent struct {
	Sequence    int64           `json:"sequence"`
	Type        string          `json:"type"`
	EndlyType   string          `json:"endlyType,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Value       interface{}     `json:"value,omitempty"`
	Messages    []*msg.Message  `json:"messages,omitempty"`
	Activity    *model.Activity `json:"activity,omitempty"`
	ActivityEnd bool            `json:"activityEnd,omitempty"`
}

type GetOperationRequest struct {
	SessionID   string `json:"sessionId"`
	OperationID string `json:"operationId"`
}

type ListOperationsResponse struct {
	SessionID  string       `json:"sessionId"`
	Operations []*Operation `json:"operations"`
}

type ListOperationsRequest struct {
	SessionID string `json:"sessionId"`
}

type StopOperationRequest = GetOperationRequest

type InspectContextRequest struct {
	SessionID   string `json:"sessionId"`
	OperationID string `json:"operationId,omitempty"`
	Full        bool   `json:"full,omitempty"`
	Path        string `json:"path,omitempty"`
}

type ContextInspection struct {
	SessionID string      `json:"sessionId"`
	Path      string      `json:"path,omitempty"`
	Found     bool        `json:"found"`
	Value     interface{} `json:"value,omitempty"`
}

type SetLoggingRequest struct {
	SessionID string `json:"sessionId"`
	Enabled   bool   `json:"enabled"`
}

type GetLoggingRequest struct {
	SessionID string `json:"sessionId"`
}

type LoggingState struct {
	SessionID string `json:"sessionId"`
	Enabled   bool   `json:"enabled"`
}

type DebugCommandRequest struct {
	SessionID   string      `json:"sessionId"`
	OperationID string      `json:"operationId"`
	Command     string      `json:"command"`
	Breakpoint  *debug.Step `json:"breakpoint,omitempty"`
}

type DebugState = debug.State
