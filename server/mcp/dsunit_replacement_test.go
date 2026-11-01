package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/dsunit"
)

// Generic file-only batches: no datastore registration or business services.
func TestMCPTableRecordsReplaceBetweenOperations(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	handler, err := NewHandler(context.Background(), runtime, endly.New)
	require.NoError(t, err)
	session := call(t, handler, "endly_open", nil)["sessionId"].(string)
	batches := []struct{ name, body string }{
		{"first", `[{"Table":"EXAMPLE_RECORD","Key":"first","Data":[{"ID":11,"VALUE":"first"}]},{"Table":"OLD_ONLY","Key":"old","Data":[{"COUNT":1}]}]`},
		{"second", `[{"Table":"EXAMPLE_RECORD","Key":"second","Data":[{"ID":22,"VALUE":"second"}]}]`},
	}
	for _, batch := range batches {
		filename := filepath.Join(t.TempDir(), batch.name+".json")
		require.NoError(t, os.WriteFile(filename, []byte(batch.body), 0600))
		source := fmt.Sprintf("init:\n  groupExpected: {global: {}}\npipeline:\n  replace:\n    action: nop\n    post:\n      groupExpected.global.example: $LoadJSON('%s')\n  convert:\n    action: nop\n    post:\n      nativeExpectData: $AsTableRecords('groupExpected.global.example')\n", filepath.ToSlash(filename))
		call(t, handler, "endly_loadWorkflow", map[string]interface{}{"sessionId": session, "url": batch.name + ".yaml", "alias": batch.name, "content": source})
		operation := call(t, handler, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": batch.name})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		done, err := runtime.WaitOperation(ctx, session, operation["id"].(string))
		cancel()
		require.NoError(t, err)
		require.Equal(t, manager.OperationSucceeded, done.Status, done.Error)
	}
	source := call(t, handler, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "groupExpected.global.example"})
	require.Len(t, source["value"], 1, "the second source really replaced the first batch")
	converted := call(t, handler, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "nativeExpectData"})
	expected := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(`{"EXAMPLE_RECORD":[{"ID":22,"VALUE":"second"}]}`), &expected))
	require.Equal(t, expected, converted["value"], "conversion must not retain a prior operation's rows or logical datasets")
}

func TestMCPTableRecordsIncludeCollectionAppendedBetweenOperations(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	handler, err := NewHandler(context.Background(), runtime, endly.New)
	require.NoError(t, err)
	session := call(t, handler, "endly_open", nil)["sessionId"].(string)
	for i, body := range []string{
		`{"Table":"EXAMPLE_RECORD","Key":"first","Data":[{"ID":11}]}`,
		`{"Table":"NEW_ONLY","Key":"second","Data":[{"ID":22}]}`,
	} {
		filename := filepath.Join(t.TempDir(), "record.json")
		require.NoError(t, os.WriteFile(filename, []byte(body), 0600))
		name := fmt.Sprintf("append%d", i)
		source := fmt.Sprintf("pipeline:\n  append:\n    action: nop\n    post:\n      ->groupExpected.global.example: $LoadJSON('%s')\n  convert:\n    action: nop\n    post:\n      nativeExpectData: $AsTableRecords('groupExpected.global.example')\n", filepath.ToSlash(filename))
		call(t, handler, "endly_loadWorkflow", map[string]interface{}{"sessionId": session, "url": name + ".yaml", "alias": name, "content": source})
		operation := call(t, handler, "endly_runWorkflow", map[string]interface{}{"sessionId": session, "workflow": name})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		done, err := runtime.WaitOperation(ctx, session, operation["id"].(string))
		cancel()
		require.NoError(t, err)
		require.Equal(t, manager.OperationSucceeded, done.Status, done.Error)
	}
	converted := call(t, handler, "endly_inspectContext", map[string]interface{}{"sessionId": session, "path": "nativeExpectData"})
	expected := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(`{"EXAMPLE_RECORD":[{"ID":11}],"NEW_ONLY":[{"ID":22}]}`), &expected))
	require.Equal(t, expected, converted["value"], "a native append to the same collection must invalidate its frozen conversion shape")
}
