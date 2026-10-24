package manager

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/service/testing/runner/http"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type emptyRolesInspection struct{ Path string }

func TestLoadedJSONEmptyContainerWireCapture(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "body.json"), []byte(`{"data":[]}`), 0600))
	var wire string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		wire = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	service := New(endly.New)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(`init:
  loaded: $LoadJSON('`+filepath.Join(root, "body.json")+`')
pipeline:
  send:
    action: http/runner:send
    requests:
      - Method: PATCH
        URL: `+server.URL+`
        JSONBody: $loaded
        Expect:
          Code: 200
`), 0600))
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: "pipeline:\n  nested:\n    action: run\n    request: '@child'\n"})
	require.NoError(t, err)
	op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root"})
	require.NoError(t, err)
	op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, op.Status, op.Error)
	t.Logf("CAPTURED_SYNTHETIC_BODY=%s", wire)
	require.JSONEq(t, `{"data":[]}`, wire, "an authored empty array must remain an array on the HTTP wire")
}

func TestLoadedJSONEmptyRolesThroughNestedWorkflowHTTPExpectation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "bootstrap-inputs.json"), []byte(`{"viewer":{"roles":[]}}`), 0600))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"roles":[]}}`))
	}))
	defer server.Close()
	require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(`init:
  identityFixture: $params.identityFixture
pipeline:
  inspectFixture:
    action: inspect-empty-roles:check
    Path: identityFixture.roles
  userInfo:
    action: http/runner:send
    requests:
      - Method: GET
        URL: `+server.URL+`
        Expect:
          Code: 200
          JSONBody:
            info:
              roles: $identityFixture.roles
`), 0600))
	observed := make(chan string, 2)
	factory := func() endly.Manager {
		engine := endly.New()
		inspector := endly.NewAbstractService("inspect-empty-roles")
		inspector.Register(&endly.Route{Action: "check", RequestProvider: func() interface{} { return &emptyRolesInspection{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			path := request.(*emptyRolesInspection).Path
			state := ctx.State()
			value, found := state.GetValue(path)
			roles, ok := value.([]interface{})
			if !found || !ok || len(roles) != 0 {
				return nil, fmt.Errorf("%s was not an empty list: %T", path, value)
			}
			observed <- path
			return &struct{}{}, nil
		}})
		engine.Register(inspector)
		return engine
	}
	manager := New(factory)
	defer manager.Shutdown(context.Background())
	session, err := manager.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = manager.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: `pipeline:
  bindings:
    action: nop
    init:
      bootstrapIdentities: $LoadJSON('` + filepath.Join(root, "bootstrap-inputs.json") + `')
  inspectBindings:
    action: inspect-empty-roles:check
    Path: bootstrapIdentities.viewer.roles
  viewer:
    action: run
    request: '@child'
    sharedState: true
    params:
      identityFixture: $bootstrapIdentities.viewer
`})
	require.NoError(t, err)
	op, err := manager.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root"})
	require.NoError(t, err)
	op, err = manager.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, op.Status, op.Error)
	require.Equal(t, 2, len(observed))
	require.Equal(t, "bootstrapIdentities.viewer.roles", <-observed)
	require.Equal(t, "identityFixture.roles", <-observed)
}
