package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

type capturedQuery struct{ SQL string }

func TestLoadedJSONIndirectTaskInitExpression(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "body.json"), []byte(`{"data":[{"id":"$AsInt(${agency2.ID})"}]}`), 0600))
	var captured string
	factory := func() endly.Manager {
		m := endly.New()
		svc := endly.NewAbstractService("capture")
		svc.Register(&endly.Route{Action: "query", RequestProvider: func() interface{} { return &capturedQuery{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, r interface{}) (interface{}, error) {
			captured = r.(*capturedQuery).SQL
			return &struct{}{}, nil
		}})
		m.Register(svc)
		return m
	}
	service := New(factory)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(`init:
  agency2:
    ID: $params.fixtureID
  targetInvalidUpdate: $LoadJSON('`+filepath.Join(root, "body.json")+`')
pipeline:
  check:
    action: capture:query
    init:
      id: ${targetInvalidUpdate.data[0].id}
    SQL: SELECT ID = $id
`), 0600))
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: `pipeline:
  nested:
    action: run
    request: '@child'
`})
	require.NoError(t, err)
	for _, id := range []int{9973, 10021} {
		captured = ""
		op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root", Params: map[string]interface{}{"fixtureID": id}})
		require.NoError(t, err)
		op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
		require.Equal(t, fmt.Sprintf("SELECT ID = %d", id), captured)
	}
}
