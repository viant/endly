package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
)

func call(t *testing.T, h *protocol.DefaultHandler, name string, args map[string]interface{}) map[string]interface{} {
	t.Helper()
	entry, ok := h.ToolRegistry.Get(name)
	if !ok {
		t.Fatalf("missing tool %s", name)
	}
	req := &schema.CallToolRequest{}
	req.Params.Arguments = args
	result, rpcErr := entry.Handler(context.Background(), req)
	if rpcErr != nil {
		t.Fatalf("%s: %v", name, rpcErr)
	}
	text := result.Content[0].(schema.TextContent).Text
	if result.IsError != nil && *result.IsError {
		t.Fatalf("%s: %s", name, text)
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMCPControlLifecycleAndSkills(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	h, err := NewHandler(context.Background(), runtime, endly.New)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range runtime.Actions() {
		if _, ok := h.ToolRegistry.Get("endly_" + action); !ok {
			t.Fatalf("unexposed action %s", action)
		}
	}
	catalog := h.ListRegisteredSkills()
	if len(catalog) < 50 {
		t.Fatalf("incomplete catalog: %d", len(catalog))
	}
	skill := call(t, h, "endly_skill_get", map[string]interface{}{"uri": "endly://skills/endly-dsunit/SKILL.md"})
	if len(skill["contents"].([]interface{})) != 1 {
		t.Fatal("entrypoint missing")
	}
	call(t, h, "endly_resource_read", map[string]interface{}{"uri": "endly://skills/endly-dsunit/references/service.md"})
	session := call(t, h, "endly_open", map[string]interface{}{})["sessionId"].(string)
	call(t, h, "endly_loadWorkflow", map[string]interface{}{"sessionId": session, "url": "sample.yaml", "alias": "sample", "content": `pipeline:
  group:
    init:
      parent: retained
    first:
      action: nop
      init:
        value: first
    second:
      action: nop
      init:
        value: second
`})
	tree := call(t, h, "endly_listTasks", map[string]interface{}{"sessionId": session, "workflow": "sample"})
	if len(tree["tasks"].([]interface{})) != 1 {
		t.Fatal("task tree missing")
	}
	op := call(t, h, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": "sample", "tasks": "group.second,group.first", "selectorMode": "path"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done, err := runtime.WaitOperation(ctx, session, op["id"].(string))
	if err != nil || done.Status != manager.OperationSucceeded {
		t.Fatalf("run failed: %+v %v", done, err)
	}
	state := call(t, h, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "value"})
	if state["value"] != "first" {
		t.Fatalf("selector order lost: %v", state)
	}
	parent := call(t, h, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "parent"})
	if parent["value"] != "retained" {
		t.Fatal("ancestor init lost")
	}
	summary := call(t, h, "endly_getOperation", map[string]interface{}{"sessionId": session, "operationId": done.ID})
	full := call(t, h, "endly_getOperation", map[string]interface{}{"sessionId": session, "operationId": done.ID, "detail": true})
	conciseBytes, _ := json.Marshal(summary)
	fullBytes, _ := json.Marshal(full)
	if len(conciseBytes)*3 >= len(fullBytes) {
		t.Fatal("operation summary is not substantially smaller")
	}
	page := call(t, h, "endly_listOperations", map[string]interface{}{"sessionId": session, "filter": "succeeded", "limit": 1})
	if len(page["items"].([]interface{})) != 1 {
		t.Fatal("operation filter/paging failed")
	}
	events := call(t, h, "endly_operation_events", map[string]interface{}{"sessionId": session, "operationId": done.ID, "limit": 2})
	if len(events["events"].([]interface{})) != 2 || !events["hasMore"].(bool) {
		t.Fatal("event paging failed")
	}
	next := call(t, h, "endly_operation_events", map[string]interface{}{"sessionId": session, "operationId": done.ID, "after": events["nextAfter"], "limit": 2})
	if next["nextAfter"].(float64) <= events["nextAfter"].(float64) {
		t.Fatal("event cursor did not advance")
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/endly/skills/endly-dsunit-sequences", nil)
	rec := httptest.NewRecorder()
	SkillsHTTP(h)(rec, req)
	if rec.Code != 200 {
		t.Fatalf("skills API: %d", rec.Code)
	}
	call(t, h, "endly_close", map[string]interface{}{"sessionId": session})
}

func TestMCPDebugStateAndResume(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	h, err := NewHandler(context.Background(), runtime, endly.New)
	if err != nil {
		t.Fatal(err)
	}
	session := call(t, h, "endly_open", nil)["sessionId"].(string)
	call(t, h, "endly_loadWorkflow", map[string]interface{}{"sessionId": session, "url": "debug.yaml", "content": `pipeline:
  first:
    action: nop
    init:
      value: first
  second:
    action: nop
    init:
      value: second
`})
	op := call(t, h, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": "debug", "debug": true})
	id := op["id"].(string)
	waitPaused := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			state, err := runtime.DebugState(session, id)
			if err == nil && state.Status == "paused" {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("debugger did not pause")
	}
	waitPaused()
	call(t, h, "endly_getDebugState", map[string]interface{}{"sessionId": session, "operationId": id})
	call(t, h, "endly_inspectContext", map[string]interface{}{"sessionId": session, "operationId": id, "full": true})
	call(t, h, "endly_debugCommand", map[string]interface{}{"sessionId": session, "operationId": id, "command": "next"})
	waitPaused()
	call(t, h, "endly_debugCommand", map[string]interface{}{"sessionId": session, "operationId": id, "command": "continue"})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done, err := runtime.WaitOperation(ctx, session, id)
	if err != nil || done.Status != manager.OperationSucceeded {
		t.Fatalf("resume failed: %v %v", done, err)
	}
	state := call(t, h, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "value"})
	if state["value"] != "second" {
		t.Fatal("wrong final state")
	}
}

func TestLocalProjectSkillIsExplicit(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	source := fstest.MapFS{"SKILL.md": &fstest.MapFile{Data: []byte("---\nname: local-project\ndescription: Local project orchestration guidance.\n---\nLoad the project workflow before running it.\n")}}
	h, err := NewHandler(context.Background(), runtime, endly.New, source)
	if err != nil {
		t.Fatal(err)
	}
	call(t, h, "endly_skill_get", map[string]interface{}{"uri": "endly://skills/local-project/SKILL.md"})
	if _, err := NewHandler(context.Background(), runtime, endly.New, source, source); err == nil {
		t.Fatal("duplicate local skill was accepted")
	}
}
