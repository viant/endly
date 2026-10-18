package manager

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"os"
	"path/filepath"
	"testing"
)

type capturedContainerQuery struct {
	SQL    string
	Expect interface{}
}

func TestLoadedJSONContainerRuntimeExpansion(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "body.json"), []byte(`{"data":[{"id":"$AsInt(${agency2.ID})","startTime":"$FormatTime($params.startDate, 'yyyy-MM-dd hh:mm:ss')","literal":"ordinary text"}]}`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "expect.json"), []byte(`{"STARTDATE":"$FormatTime(${body.data[0].startTime}, 'yyyy-MM-dd hh:mm:ss')","ID":"$AsInt(${agency2.ID})","literal":"ordinary text"}`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "list.json"), []byte(`[{"id":"$AsInt(${agency2.ID})","literal":"ordinary text"}]`), 0600))
	var captured capturedContainerQuery
	factory := func() endly.Manager {
		m := endly.New()
		svc := endly.NewAbstractService("capture")
		svc.Register(&endly.Route{Action: "query", RequestProvider: func() interface{} { return &capturedContainerQuery{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, r interface{}) (interface{}, error) {
			captured = *r.(*capturedContainerQuery)
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
  body: $LoadJSON('`+filepath.Join(root, "body.json")+`')
  expected: $LoadJSON('`+filepath.Join(root, "expect.json")+`')
  list: $LoadJSON('`+filepath.Join(root, "list.json")+`')
pipeline:
  check:
    action: capture:query
    init:
      id: ${body.data[0].id}
    SQL: SELECT ID = $id
    Expect:
      object: $expected
      list: $list
`), 0600))
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: "pipeline:\n  nested:\n    action: run\n    request: '@child'\n"})
	require.NoError(t, err)
	for i, id := range []int{9973, 10021} {
		date := fmt.Sprintf("2030-01-%02d 12:00:00", i+10)
		op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root", Params: map[string]interface{}{"fixtureID": id, "startDate": date}})
		require.NoError(t, err)
		op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
		require.Equal(t, fmt.Sprintf("SELECT ID = %d", id), captured.SQL)
		expected := captured.Expect.(map[string]interface{})
		object := expected["object"].(map[string]interface{})
		require.Equal(t, date, object["STARTDATE"])
		require.EqualValues(t, id, object["ID"])
		require.Equal(t, "ordinary text", object["literal"])
		list := expected["list"].([]interface{})
		require.EqualValues(t, id, list[0].(map[string]interface{})["id"])
	}
}
