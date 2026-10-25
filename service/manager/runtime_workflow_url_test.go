package manager

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"os"
	"path/filepath"
	"testing"
)

type runtimeURLCapture struct{ Value string }

func TestRuntimeWorkflowURLMaterializedAfterParentLoad(t *testing.T) {
	root := t.TempDir()
	var captured string
	factory := func() endly.Manager {
		m := endly.New()
		capture := endly.NewAbstractService("runtimeCapture")
		capture.Register(&endly.Route{Action: "value", RequestProvider: func() interface{} { return &runtimeURLCapture{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, request interface{}) (interface{}, error) {
			captured = request.(*runtimeURLCapture).Value
			return &struct{}{}, nil
		}})
		m.Register(capture)
		return m
	}
	service := New(factory)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "parent", URL: filepath.Join(root, "parent.yaml"), Format: "yaml", Content: "pipeline:\n  generated:\n    action: workflow:run\n    URL: $params.child\n    params:\n      value: $params.value\n"})
	require.NoError(t, err)
	// Neither the workflow nor its relative request/default exists at parent load.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "generated", "default"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "generated", "default", "payload.yaml"), []byte("Value: $params.value\n"), 0600))
	for index, name := range []string{"parent", "second"} {
		child := filepath.Join(root, "generated", name+".yaml")
		require.NoError(t, os.WriteFile(child, []byte("pipeline:\n  capture:\n    action: runtimeCapture:value\n    request: '@payload'\n"), 0600))
		value := []string{"fresh-first", "fresh-second"}[index]
		op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "parent", Params: map[string]interface{}{"child": child, "value": value}})
		require.NoError(t, err)
		op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
		require.Equal(t, value, captured)
		runtimeSession, lookupErr := service.lookup(session.SessionID)
		require.NoError(t, lookupErr)
		registered, lookupErr := runtimeSession.workflow.Workflow("parent")
		require.NoError(t, lookupErr)
		require.Contains(t, registered.Source.URL, filepath.ToSlash(filepath.Join(root, "parent.yaml")))
	}
	missing := filepath.Join(root, "missing.yaml")
	op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "parent", Params: map[string]interface{}{"child": missing, "value": "unexecuted"}})
	require.NoError(t, err)
	op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationFailed, op.Status)
	require.Contains(t, op.Error, "missing.yaml")
	require.Equal(t, "fresh-second", captured)
}
