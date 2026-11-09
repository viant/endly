package http_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/service/testing/runner/http"
	"github.com/viant/endly/service/workflow"
)

func TestWorkflowHTTPEmptyObjectWireIdentity(t *testing.T) {
	captured := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		captured <- string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	source := fmt.Sprintf(`init:
  adorder:
    ID: 7
pipeline:
  testFrequencyCappingSparseContract:
    action: http/runner:send
    init:
      adOrderId: ${adorder.ID}
    requests:
      - Method: PATCH
        URL: %s
        JSONBody:
          data:
            - "@type": std
              adOrderId: $AsInt(${adOrderId})
              frequencyCapping: {}
              roles: []
              nullable: null
              nested:
                object: {}
                list: []
        Expect:
          Code: 200
          JSONBody:
            status: ok
`, server.URL)
	dir := t.TempDir()
	path := filepath.Join(dir, "run.yaml")
	child := filepath.Join(dir, "cases", "048_audience_patch")
	require.NoError(t, os.MkdirAll(child, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(child, "test.yaml"), []byte(source), 0600))
	source = `pipeline:
  cases:
    tag: $pathMatch
    subPath: cases/${index}_*
    template:
      testWS:
        action: run
        init:
          parentPath: $path
          tag: $tag
        request: "@test"
`
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	manager := endly.New()
	manager.Register(workflow.New())
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	request, err := workflow.NewRunRequestFromURL(path)
	require.NoError(t, err)
	request.AssetURL, request.Name, request.SharedState = path, "run", true
	var response workflow.RunResponse
	require.NoError(t, endly.Run(ctx, request, &response))
	select {
	case body := <-captured:
		require.JSONEq(t, `{"data":[{"@type":"std","adOrderId":7,"frequencyCapping":{},"roles":[],"nullable":null,"nested":{"object":{},"list":[]}}]}`, body)
	default:
		t.Fatal("HTTP request was not executed")
	}
}
