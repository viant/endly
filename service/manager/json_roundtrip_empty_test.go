package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

type jsonRoundtripCapture struct{ Value string }

func TestLoadedJSONSerializationPreservesEmptyContainers(t *testing.T) {
	root := t.TempDir()
	before := `{"settings":{"items":[],"object":{},"nested":[{"items":[]}],"enabled":false,"zero":0}}`
	require.NoError(t, os.WriteFile(filepath.Join(root, "input.json"), []byte(before), 0600))
	var captured []string
	factory := func() endly.Manager {
		engine := endly.New()
		capture := endly.NewAbstractService("capture-json-shape")
		capture.Register(&endly.Route{Action: "read", RequestProvider: func() interface{} { return &jsonRoundtripCapture{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, request interface{}) (interface{}, error) {
			captured = append(captured, request.(*jsonRoundtripCapture).Value)
			return &struct{}{}, nil
		}})
		engine.Register(capture)
		return engine
	}
	manager := New(factory)
	defer manager.Shutdown(context.Background())
	session, err := manager.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = manager.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: `pipeline:
  load:
    action: nop
    init:
      config: $LoadJSON('` + filepath.Join(root, "input.json") + `')
  before:
    action: capture-json-shape:read
    Value: $AsJSON('$config')
  project:
    action: nop
    init:
      config.settings.label: synthetic
  after:
    action: capture-json-shape:read
    Value: $AsJSON('$config')
`})
	require.NoError(t, err)
	op, err := manager.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root"})
	require.NoError(t, err)
	op, err = manager.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, op.Status, op.Error)
	require.Len(t, captured, 2)
	require.JSONEq(t, before, captured[0])
	require.JSONEq(t, `{"settings":{"items":[],"object":{},"nested":[{"items":[]}],"enabled":false,"zero":0,"label":"synthetic"}}`, captured[1])
}
