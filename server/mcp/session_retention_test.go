package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/service/manager"
	"github.com/viant/mcp-protocol/schema"
)

func TestMCPActiveWorkflowSurvivesIdleClient(t *testing.T) {
	runtime := manager.New(endly.New, manager.WithSessionTTL(50*time.Millisecond))
	defer runtime.Shutdown(context.Background())
	h, err := NewHandler(context.Background(), runtime, endly.New)
	require.NoError(t, err)
	session := call(t, h, "endly_open", map[string]interface{}{})["sessionId"].(string)
	call(t, h, "endly_loadWorkflow", map[string]interface{}{
		"sessionId": session, "url": "retention.yaml", "alias": "retention",
		"content": "pipeline:\n  work:\n    action: nop\n",
	})
	op := call(t, h, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": "retention", "debug": true})
	args := map[string]interface{}{"sessionId": session, "operationId": op["id"]}
	require.Eventually(t, func() bool {
		return call(t, h, "endly_getOperation", args)["status"] == "paused"
	}, time.Second, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime.StartJanitor(ctx, 5*time.Millisecond)
	// A real workflow is held at its native debug boundary. No client activity
	// renews the session while several idle TTLs and janitor ticks pass.
	time.Sleep(150 * time.Millisecond)
	require.Equal(t, "paused", call(t, h, "endly_getOperation", args)["status"])
	call(t, h, "endly_debugCommand", map[string]interface{}{"sessionId": session, "operationId": op["id"], "command": "continue"})
	require.Eventually(t, func() bool {
		return call(t, h, "endly_getOperation", args)["status"] == manager.OperationSucceeded
	}, time.Second, time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	entry, ok := h.ToolRegistry.Get("endly_getOperation")
	require.True(t, ok)
	req := &schema.CallToolRequest{}
	req.Params.Arguments = args
	result, rpcErr := entry.Handler(context.Background(), req)
	require.Nil(t, rpcErr)
	require.NotNil(t, result)
	require.NotNil(t, result.IsError)
	require.True(t, *result.IsError, "completed idle sessions still expire through the MCP error boundary")
}
