package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/dsunit"
	_ "github.com/viant/endly/service/testing/endpoint/http"
	_ "github.com/viant/endly/service/testing/log"
	_ "github.com/viant/endly/service/testing/runner/http"
	_ "github.com/viant/endly/service/testing/validator"
)

func TestNativeSourceToolsDispatchAndShareState(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	h, err := NewHandler(context.Background(), runtime, endly.New)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"endly_dsunit_mapping", "endly_dsunit_sequence", "endly_dsunit_prepare", "endly_dsunit_expect", "endly_http_endpoint_listen", "endly_http_endpoint_append", "endly_http_endpoint_shutdown", "endly_http_runner_send", "endly_validator_assert", "endly_validator_log_assert"} {
		if _, ok := h.ToolRegistry.Get(name); !ok {
			t.Fatalf("missing native tool %s", name)
		}
	}
	entry, _ := h.ToolRegistry.Get("endly_http_runner_send")
	request := entry.Metadata.InputSchema.Properties["request"]
	if request["type"] != "object" || request["properties"] == nil {
		t.Fatal("native request schema missing")
	}
	session := call(t, h, "endly_open", nil)["sessionId"].(string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"item-1"}`))
	}))
	defer server.Close()
	op := call(t, h, "endly_http_runner_send", map[string]interface{}{"sessionId": session, "request": map[string]interface{}{"Requests": []interface{}{map[string]interface{}{"Method": "GET", "URL": server.URL, "Expect": map[string]interface{}{"Code": 200}}}}})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done, err := runtime.WaitOperation(ctx, session, op["id"].(string))
	if err != nil || done.Status != manager.OperationSucceeded {
		t.Fatalf("native send failed: %+v %v", done, err)
	}
	if done.PassedAssertions == 0 {
		t.Fatal("native assertions were not recorded")
	}
	failure := call(t, h, "endly_validator_assert", map[string]interface{}{"sessionId": session, "request": map[string]interface{}{"Expect": "expected", "Actual": "wrong"}})
	done, err = runtime.WaitOperation(ctx, session, failure["id"].(string))
	if err != nil || done.Status != manager.OperationFailed || done.FailedAssertions == 0 {
		t.Fatalf("native validation failure lost: %+v %v", done, err)
	}
}
