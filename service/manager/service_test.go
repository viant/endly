package manager

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

type callbackTransport struct{ count atomic.Int64 }

func (c *callbackTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	c.count.Add(1)
	return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}

func TestServiceSessionWorkflowLifecycle(t *testing.T) {
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{Name: "test"})
	require.NoError(t, err)
	require.NotEmpty(t, session.SessionID)

	loaded, err := service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{
		SessionID: session.SessionID,
		URL:       "test.yaml",
		Alias:     "sample",
		Content: `pipeline:
  first:
    action: nop
`,
		Format: "yaml",
	})
	require.NoError(t, err)
	require.Equal(t, session.SessionID, loaded.SessionID)
	require.Equal(t, "sample", loaded.Alias)
	require.Equal(t, []string{"first"}, loaded.Tasks)

	operation, err := service.StartWorkflow(&RunWorkflowRequest{
		SessionID: session.SessionID,
		Workflow:  "sample",
		Tasks:     "*",
		Params:    map[string]interface{}{"answer": 42},
	})
	require.NoError(t, err)
	operation, err = service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, operation.Status)
	require.NotEmpty(t, operation.Events)

	inspection, err := service.InspectContext(&InspectContextRequest{SessionID: session.SessionID, Path: "answer"})
	require.NoError(t, err)
	require.True(t, inspection.Found)
	require.EqualValues(t, 42, inspection.Value)

	logging, err := service.SetLogging(&SetLoggingRequest{SessionID: session.SessionID, Enabled: false})
	require.NoError(t, err)
	require.False(t, logging.Enabled)

	require.NoError(t, service.UnloadWorkflow(&UnloadWorkflowRequest{SessionID: session.SessionID, Workflow: "sample"}))
	workflows, err := service.ListWorkflows(session.SessionID)
	require.NoError(t, err)
	require.Empty(t, workflows.Workflows)
	require.NoError(t, service.Close(context.Background(), session.SessionID))
}

func TestServiceStopOperation(t *testing.T) {
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{
		SessionID: session.SessionID,
		URL:       "slow.yaml",
		Content: `pipeline:
  wait:
    action: nop
    sleepTimeMs: 5000
`,
		Format: "yaml",
	})
	require.NoError(t, err)
	operation, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "slow", Tasks: "*"})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		current, getErr := service.GetOperation(session.SessionID, operation.ID)
		return getErr == nil && current.Status == OperationRunning
	}, time.Second, 10*time.Millisecond)
	_, err = service.StopOperation(session.SessionID, operation.ID)
	require.NoError(t, err)
	finished, err := service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationCancelled, finished.Status)
}

func TestServiceDebugWorkflow(t *testing.T) {
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{
		SessionID: session.SessionID,
		URL:       "debug.yaml",
		Content: `pipeline:
  first:
    action: nop
`,
		Format: "yaml",
	})
	require.NoError(t, err)
	operation, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "debug", Tasks: "*", Debug: true})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		state, stateErr := service.DebugState(session.SessionID, operation.ID)
		return stateErr == nil && state.Status == "paused" && state.Point.Kind == "task"
	}, time.Second, 10*time.Millisecond)
	_, err = service.DebugCommand(&DebugCommandRequest{SessionID: session.SessionID, OperationID: operation.ID, Command: "step"})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		state, stateErr := service.DebugState(session.SessionID, operation.ID)
		return stateErr == nil && state.Status == "paused" && state.Point.Kind == "action"
	}, time.Second, 10*time.Millisecond)
	inspection, err := service.InspectContext(&InspectContextRequest{SessionID: session.SessionID, OperationID: operation.ID, Full: true})
	require.NoError(t, err)
	require.True(t, inspection.Found)
	_, err = service.DebugCommand(&DebugCommandRequest{SessionID: session.SessionID, OperationID: operation.ID, Command: "continue"})
	require.NoError(t, err)
	finished, err := service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, finished.Status)
}

func TestServiceRunAction(t *testing.T) {
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	operation, err := service.StartAction(&RunActionRequest{
		SessionID: session.SessionID,
		Service:   "workflow",
		Action:    "nop",
		Request:   map[string]interface{}{},
	})
	require.NoError(t, err)
	finished, err := service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, finished.Status)
	require.Equal(t, "action", finished.Kind)
}

func TestServiceWorkflowCallbacks(t *testing.T) {
	transport := &callbackTransport{}
	service := New(endly.New,
		WithCallbackHosts("callback.test"),
		WithCallbackHTTPClient(&http.Client{Transport: transport}),
	)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, URL: "callback.yaml", Content: "pipeline:\n  first:\n    action: nop\n", Format: "yaml"})
	require.NoError(t, err)
	operation, err := service.StartWorkflow(&RunWorkflowRequest{
		SessionID: session.SessionID,
		Workflow:  "callback",
		Tasks:     "*",
		Callbacks: []Callback{{URL: "https://callback.test/endly"}},
	})
	require.NoError(t, err)
	_, err = service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return transport.count.Load() >= 2 }, time.Second, 10*time.Millisecond)
}

func TestServiceActionPolicyAppliesToNestedActions(t *testing.T) {
	service := New(endly.New, WithAllowedActions("workflow:print"))
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, URL: "policy.yaml", Content: "pipeline:\n  denied:\n    action: nop\n", Format: "yaml"})
	require.NoError(t, err)
	operation, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "policy"})
	require.NoError(t, err)
	finished, err := service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationFailed, finished.Status)
	require.Contains(t, finished.Error, "workflow:nop was not allowed")
	_, err = service.StartAction(&RunActionRequest{SessionID: session.SessionID, Service: "workflow", Action: "nop"})
	require.ErrorContains(t, err, "workflow:nop was not allowed")
}

func TestServiceLimitsAndRetention(t *testing.T) {
	service := New(endly.New, WithMaxSessions(1), WithOperationRetention(time.Millisecond))
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.Open(context.Background(), &OpenRequest{})
	require.ErrorContains(t, err, "session limit")
	operation, err := service.StartAction(&RunActionRequest{SessionID: session.SessionID, Service: "workflow", Action: "nop"})
	require.NoError(t, err)
	_, err = service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	service.cleanup(time.Now().UTC().Add(time.Second))
	_, err = service.GetOperation(session.SessionID, operation.ID)
	require.ErrorContains(t, err, "was not found")
}

func TestAllowedActionsCommaSeparated(t *testing.T) {
	service := New(endly.New, WithAllowedActions("workflow:print, workflow:nop", "exec:run"))
	require.NoError(t, service.authorizeAction("workflow", "print"))
	require.NoError(t, service.authorizeAction("workflow", "nop"))
	require.NoError(t, service.authorizeAction("exec", "run"))
	require.Error(t, service.authorizeAction("storage", "copy"))
}

func TestServiceRunWorkflowPreservesSelectedTaskOrderAndDuplicates(t *testing.T) {
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{
		SessionID: session.SessionID,
		URL:       "ordered.yaml",
		Content: `pipeline:
  task1:
    action: nop
  task2:
    action: nop
  task3:
    action: nop
`,
		Format: "yaml",
	})
	require.NoError(t, err)
	operation, err := service.StartWorkflow(&RunWorkflowRequest{
		SessionID: session.SessionID,
		Workflow:  "ordered",
		Tasks:     "task1,task2,task1",
	})
	require.NoError(t, err)
	finished, err := service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, finished.Status)
	var actual []string
	for _, event := range finished.Events {
		if event.Type != "task.started" {
			continue
		}
		value, ok := event.Value.(map[string]interface{})
		if ok {
			actual = append(actual, value["TaskName"].(string))
		}
	}
	require.Equal(t, []string{"task1", "task2", "task1"}, actual)
}
