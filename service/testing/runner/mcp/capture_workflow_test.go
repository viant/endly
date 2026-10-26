package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/runner/mcp"
)

func TestReadCaptureLoadedWorkflowUsesFilteredResponsePaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "browser.jsonl")
	var lines []byte
	for _, window := range []string{"campaign", "advertiser"} {
		request, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": map[string]interface{}{"name": "namespaces/preview", "arguments": map[string]interface{}{"resource": map[string]interface{}{"uri": "window://platform/" + window}}}})
		response, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": 7, "result": map[string]interface{}{"structuredContent": map[string]interface{}{"sessionId": window + "-session"}}})
		event, err := json.Marshal(map[string]interface{}{"type": "network", "tx": map[string]interface{}{"Status": 200, "RequestBody": map[string]interface{}{"Data": string(request)}, "ResponseBody": map[string]interface{}{"Data": string(response)}}})
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, append(event, '\n')...)
	}
	if err := os.WriteFile(file, lines, 0600); err != nil {
		t.Fatal(err)
	}
	svc := manager.New(func() endly.Manager { return endly.New() })
	defer svc.Shutdown(context.Background())
	session, err := svc.Open(context.Background(), &manager.OpenRequest{})
	if err != nil {
		t.Fatal(err)
	}
	definition := `pipeline:
  captured:
    action: mcp/runner:readCapture
    source: {URL: '` + filepath.ToSlash(file) + `'}
    methods: [namespaces/preview]
    arguments: {resource: {uri: 'window://platform/advertiser'}}
  count:
    action: validator:assert
    actual: $Len($captured.Transactions)
    expect: 1
  bindings:
    action: nop
    init:
      - name: capturedMethod
        from: captured.Transactions[0].method
        required: true
      - name: childSession
        from: captured.Transactions[0].response.result.structuredContent.sessionId
        required: true
  method:
    action: validator:assert
    actual: $capturedMethod
    expect: namespaces/preview
  child:
    action: validator:assert
    actual: $childSession
    expect: advertiser-session
  required:
    when: $count.FailedCount > 0 || $method.FailedCount > 0 || $child.FailedCount > 0
    action: workflow:fail
    message: Native capture response paths lost the exact filtered child session
`
	_, err = svc.LoadWorkflow(context.Background(), &manager.LoadWorkflowRequest{SessionID: session.SessionID, Alias: "capture", URL: filepath.Join(dir, "workflow.yaml"), Format: "yaml", Content: definition})
	if err != nil {
		t.Fatal(err)
	}
	op, err := svc.StartWorkflow(&manager.RunWorkflowRequest{SessionID: session.SessionID, Workflow: "capture"})
	if err != nil {
		t.Fatal(err)
	}
	op, err = svc.WaitOperation(context.Background(), session.SessionID, op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if op.Status != manager.OperationSucceeded {
		for _, path := range []string{"captured", "count", "method", "child"} {
			state, inspectErr := svc.InspectContext(&manager.InspectContextRequest{SessionID: session.SessionID, Path: path})
			if inspectErr == nil {
				t.Logf("workflow keys/types only %s: %v", path, captureWorkflowShape(state.Value, 0))
			}
		}
		t.Fatalf("workflow status=%s error=%v", op.Status, op.Error)
	}
	if op.PassedAssertions < 3 || op.FailedAssertions != 0 {
		t.Fatalf("assertions passed=%d failed=%d", op.PassedAssertions, op.FailedAssertions)
	}
}

func captureWorkflowShape(value interface{}, depth int) interface{} {
	if depth > 6 || value == nil {
		return fmt.Sprintf("%T", value)
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
		out := map[string]interface{}{}
		for _, key := range v.MapKeys() {
			out[key.String()] = captureWorkflowShape(v.MapIndex(key).Interface(), depth+1)
		}
		return out
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		out := map[string]interface{}{"type": fmt.Sprintf("%T", value), "length": v.Len()}
		if v.Len() > 0 {
			out["elementKeysAndTypes"] = captureWorkflowShape(v.Index(0).Interface(), depth+1)
		}
		return out
	}
	return fmt.Sprintf("%T", value)
}
