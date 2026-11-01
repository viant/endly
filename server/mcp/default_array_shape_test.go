package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
)

// Cover the control-plane loader and runtime child URL without preloading aliases.
// An empty default must support conditional native push, while retaining other
// native JSON shapes across the nested invocation and its parameter publication.
func TestMCPNestedWorkflowDefaultArrayShape(t *testing.T) {
	for _, inline := range []bool{false, true} {
		name := "url"
		if inline {
			name = "content"
		}
		t.Run(name, func(t *testing.T) {
			runtime := manager.New(endly.New)
			defer runtime.Shutdown(context.Background())
			handler, err := NewHandler(context.Background(), runtime, endly.New)
			require.NoError(t, err)
			root := t.TempDir()
			child := `pipeline:
  snapshot:
    action: workflow:nop
    init:
      - {name: desiredItems, from: fixtureResult.settings.items, value: []}
      - {name: disabled, from: fixtureResult.settings.disabled, value: false}
      - {name: zero, from: fixtureResult.settings.zero, value: 0}
      - {name: beforeJSON, value: '$AsJSON($desiredItems)'}
  add:
    when: '$IndexOf($desiredItems, synthetic-item) < 0'
    action: workflow:nop
    post:
      - {name: '->desiredItems', value: synthetic-item}
  capture:
    action: workflow:nop
    post:
      captured: {items: '$desiredItems', empty: [], disabled: '$disabled', zero: '$zero'}
  verify:
    action: validator:assert
    actual: $captured
    expect: {items: [synthetic-item], empty: [], disabled: false, zero: 0}
`
			require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(child), 0600))
			content := `init:
  fixtureResult: {settings: {}}
pipeline:
  nested:
    action: workflow:run
    URL: child.yaml
    sharedState: true
`
			source := filepath.Join(root, "root.yaml")
			require.NoError(t, os.WriteFile(source, []byte(content), 0600))
			session := call(t, handler, "endly_open", map[string]interface{}{})["sessionId"].(string)
			request := map[string]interface{}{"sessionId": session, "url": source, "alias": "root"}
			if inline {
				request["content"], request["format"] = content, "yaml"
			}
			call(t, handler, "endly_loadWorkflow", request)
			op := call(t, handler, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": "root"})
			done, err := runtime.WaitOperation(context.Background(), session, op["id"].(string))
			require.NoError(t, err)
			require.Equal(t, manager.OperationSucceeded, done.Status, done.Error)
			require.Zero(t, done.FailedAssertions)
			require.Positive(t, done.PassedAssertions)
			before := call(t, handler, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "beforeJSON", "detail": true})
			require.JSONEq(t, `[]`, before["value"].(string))
			inspection := call(t, handler, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "captured", "detail": true})
			encoded, err := json.Marshal(inspection["value"])
			require.NoError(t, err)
			require.JSONEq(t, `{"items":["synthetic-item"],"empty":[],"disabled":false,"zero":0}`, string(encoded))
		})
	}
}
