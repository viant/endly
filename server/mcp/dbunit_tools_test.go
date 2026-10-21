package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/dbunit"
	"github.com/viant/mcp-protocol/schema"
)

func TestDBUnitToolsShareSessionAndKeepFailedAssertions(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	handler, err := NewHandler(context.Background(), runtime, endly.New)
	require.NoError(t, err)
	for _, action := range []string{"register", "init", "script", "sql", "prepare", "expect", "query", "dump", "export", "compare"} {
		entry, ok := handler.ToolRegistry.Get("endly_dbunit_" + action)
		require.True(t, ok, action)
		request := entry.Metadata.InputSchema.Properties["request"]
		require.Equal(t, "object", request["type"])
		require.NotNil(t, request["properties"])
	}
	session := call(t, handler, "endly_open", nil)["sessionId"].(string)
	_, err = runtime.LoadWorkflow(context.Background(), &manager.LoadWorkflowRequest{SessionID: session, URL: "configure.yaml", Content: "pipeline:\n  configure:\n    action: nop\n", Format: "yaml"})
	require.NoError(t, err)
	configured, err := runtime.StartWorkflow(&manager.RunWorkflowRequest{SessionID: session, Workflow: "configure", Params: map[string]any{"connection": "app"}})
	require.NoError(t, err)
	configured, err = runtime.WaitOperation(context.Background(), session, configured.ID)
	require.NoError(t, err)
	require.Equal(t, manager.OperationSucceeded, configured.Status)
	run := func(action string, request map[string]any) *manager.Operation {
		t.Helper()
		result := call(t, handler, "endly_dbunit_"+action, map[string]any{"sessionId": session, "request": request})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		op, err := runtime.WaitOperation(ctx, session, result["id"].(string))
		require.NoError(t, err)
		return op
	}
	require.Equal(t, manager.OperationSucceeded, run("register", map[string]any{"database": "$connection", "driver": "sqlite", "dsn": filepath.Join(t.TempDir(), "app.db")}).Status)
	require.Equal(t, manager.OperationSucceeded, run("script", map[string]any{"database": "app", "statements": []any{"CREATE TABLE users(id INTEGER, name TEXT)"}}).Status)
	rows := []any{map[string]any{"id": 1, "name": "alice"}}
	require.Equal(t, manager.OperationSucceeded, run("prepare", map[string]any{"database": "app", "datasets": []any{map[string]any{"table": []any{"users"}, "source": map[string]any{"kind": "maps", "data": rows}}}}).Status)
	queried := run("query", map[string]any{"database": "app", "sql": "SELECT * FROM users", "capture": "selected"})
	require.Equal(t, manager.OperationSucceeded, queried.Status)
	state, err := runtime.InspectContext(&manager.InspectContextRequest{SessionID: session, Path: "selected"})
	require.NoError(t, err)
	require.True(t, state.Found)
	expect := func(name string) *manager.Operation {
		return run("expect", map[string]any{"database": "app", "expectations": []any{map[string]any{"table": []any{"users"}, "expected": map[string]any{"kind": "maps", "data": []any{map[string]any{"id": 1, "name": name}}}}}})
	}
	passed := expect("alice")
	require.Equal(t, manager.OperationSucceeded, passed.Status)
	require.Zero(t, passed.FailedAssertions)
	require.Positive(t, passed.PassedAssertions)
	failed := expect("wrong")
	require.Equal(t, manager.OperationFailed, failed.Status)
	require.Equal(t, 1, failed.FailedAssertions)
	require.Equal(t, passed.PassedAssertions-1, failed.PassedAssertions)
	require.NotEmpty(t, failed.Events)
	canceled := call(t, handler, "endly_dbunit_query", map[string]any{"sessionId": session, "timeoutMillis": 10, "request": map[string]any{"database": "$connection", "sql": "WITH RECURSIVE counter(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM counter WHERE x<1000000000) SELECT sum(x) FROM counter"}})
	cancelCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	finished, err := runtime.WaitOperation(cancelCtx, session, canceled["id"].(string))
	require.NoError(t, err)
	require.Equal(t, manager.OperationCancelled, finished.Status)
}

// omittedDBUnit exercises the supported minimal-registry path without global registry mutation.
type omittedDBUnit struct{ endly.Manager }

func (m *omittedDBUnit) Service(input any) (endly.Service, error) {
	if name, ok := input.(string); ok && name == "dbunit" {
		return nil, fmt.Errorf("dbunit unavailable")
	}
	return m.Manager.Service(input)
}

func TestDBUnitToolsHonorPolicyAndMissingService(t *testing.T) {
	t.Run("policy", func(t *testing.T) {
		runtime := manager.New(endly.New, manager.WithAllowedActions("dbunit:query"))
		defer runtime.Shutdown(context.Background())
		handler, err := NewHandler(context.Background(), runtime, endly.New)
		require.NoError(t, err)
		session := call(t, handler, "endly_open", nil)["sessionId"].(string)
		entry, ok := handler.ToolRegistry.Get("endly_dbunit_register")
		require.True(t, ok)
		request := &schema.CallToolRequest{}
		request.Params.Arguments = map[string]any{"sessionId": session, "request": map[string]any{"database": "app", "driver": "sqlite", "dsn": ":memory:"}}
		result, rpcErr := entry.Handler(context.Background(), request)
		require.Nil(t, rpcErr)
		require.NotNil(t, result.IsError)
		require.True(t, *result.IsError)
		require.Contains(t, result.Content[0].(schema.TextContent).Text, "dbunit:register was not allowed")
	})
	t.Run("optional service", func(t *testing.T) {
		factory := func() endly.Manager { return &omittedDBUnit{Manager: endly.New()} }
		runtime := manager.New(factory)
		defer runtime.Shutdown(context.Background())
		handler, err := NewHandler(context.Background(), runtime, factory)
		require.NoError(t, err)
		_, present := handler.ToolRegistry.Get("endly_dbunit_prepare")
		require.False(t, present)
		_, present = handler.ToolRegistry.Get("endly_dsunit_prepare")
		require.True(t, present)
	})
}

func TestDBUnitGenerationToolsComposeCapturedDatasets(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	handler, err := NewHandler(context.Background(), runtime, endly.New)
	require.NoError(t, err)
	for _, name := range []string{"hydrate", "sequence", "map"} {
		entry, present := handler.ToolRegistry.Get("endly_dbunit_" + name)
		require.True(t, present)
		encoded, err := json.Marshal(entry.Metadata.InputSchema.Properties["request"])
		require.NoError(t, err)
		require.Contains(t, string(encoded), "database")
		if name == "map" {
			require.Contains(t, string(encoded), "sourceFrom")
		}
	}
	session := call(t, handler, "endly_open", nil)["sessionId"].(string)
	run := func(action string, request map[string]any) *manager.Operation {
		t.Helper()
		result := call(t, handler, "endly_dbunit_"+action, map[string]any{"sessionId": session, "request": request})
		op, err := runtime.WaitOperation(context.Background(), session, result["id"].(string))
		require.NoError(t, err)
		require.Equal(t, manager.OperationSucceeded, op.Status, op.Error)
		return op
	}
	run("register", map[string]any{"database": "app", "driver": "sqlite", "dsn": filepath.Join(t.TempDir(), "generation.db")})
	run("script", map[string]any{"database": "app", "statements": []any{"CREATE TABLE people(id INTEGER PRIMARY KEY,name TEXT)", "INSERT INTO people VALUES(40,'seed')"}})
	run("hydrate", map[string]any{"database": "app", "capture": "generated", "sequences": []any{map[string]any{"name": "ids", "seed": map[string]any{"database": "app", "sql": "SELECT MAX(id)+1 FROM people"}}}, "tables": []any{map[string]any{"name": "people", "table": []any{"people"}, "source": map[string]any{"kind": "maps", "data": []any{map[string]any{"name": "$ToUpper(alice)"}}}, "generate": []any{map[string]any{"field": []any{"id"}, "sequence": "ids"}}}}})
	run("sequence", map[string]any{"database": "app", "capture": "continuation", "sequences": []any{map[string]any{"name": "ids", "startFrom": "generated.sequences.ids.next"}}})
	run("map", map[string]any{"database": "app", "capture": "mapped", "sourceFrom": "generated.tables[0].source", "mapping": map[string]any{"name": "people", "tables": []any{map[string]any{"name": "people", "table": "people", "keys": []any{"id"}, "columns": []any{map[string]any{"name": "id"}, map[string]any{"name": "name"}}}}}})
	run("prepare", map[string]any{"database": "app", "datasetsFrom": "mapped.datasets"})
	checked := run("expect", map[string]any{"database": "app", "expectations": []any{map[string]any{"table": []any{"people"}, "scope": "keys", "match": map[string]any{"keys": []any{"id"}}, "expected": map[string]any{"kind": "maps", "data": []any{map[string]any{"id": 41, "name": "ALICE"}}}}}})
	require.Zero(t, checked.FailedAssertions)
	require.Positive(t, checked.PassedAssertions)
	next, err := runtime.InspectContext(&manager.InspectContextRequest{SessionID: session, Path: "continuation.sequences.ids.next"})
	require.NoError(t, err)
	require.True(t, next.Found)
	require.EqualValues(t, 42, next.Value)
}
