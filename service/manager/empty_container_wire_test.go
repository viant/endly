package manager

import (
	"context"
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
	require.JSONEq(t, `{"data":{}}`, wire, "preserve original effective wire semantics")
}
