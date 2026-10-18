package manager

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"os"
	"path/filepath"
	"testing"
)

type capturedGenericRequest struct {
	Nested map[string]interface{}
	List   []interface{}
}

func TestDirectUDFInGenericRequestFile(t *testing.T) {
	root := t.TempDir()
	body := filepath.Join(root, "payload.json")
	require.NoError(t, os.WriteFile(body, []byte(`{"number":"$AsInt(${params.number})","literal":"ordinary text"}`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "request.json"), []byte(`{"Nested":{"payload":"$LoadJSON('`+body+`')"},"List":[{"payload":"$LoadJSON('`+body+`')"}]}`), 0600))
	var captured capturedGenericRequest
	factory := func() endly.Manager {
		m := endly.New()
		s := endly.NewAbstractService("capture")
		s.Register(&endly.Route{Action: "request", RequestProvider: func() interface{} { return &capturedGenericRequest{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, r interface{}) (interface{}, error) {
			captured = *r.(*capturedGenericRequest)
			return &struct{}{}, nil
		}})
		m.Register(s)
		return m
	}
	service := New(factory)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: `pipeline:
  call:
    action: capture:request
    request: '@request.json'
`})
	require.NoError(t, err)
	for _, number := range []int{17, 23} {
		op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root", Params: map[string]interface{}{"number": number}})
		require.NoError(t, err)
		op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
		require.EqualValues(t, number, captured.Nested["payload"].(map[string]interface{})["number"])
		require.EqualValues(t, number, captured.List[0].(map[string]interface{})["payload"].(map[string]interface{})["number"])
	}
}
