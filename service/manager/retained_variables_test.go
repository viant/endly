package manager

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/service/system/exec"
	_ "github.com/viant/endly/service/testing/validator"
)

func TestNestedWorkflowUsesCurrentActionOutputOnRerun(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(`pipeline:
  databaseName:
    action: exec:run
    commands:
      - printf 'database=$params.runId'
    extract:
      - key: name
        regExpr: 'database=([a-z]+)'
  check:
    action: validator:assert
    actual: root/${databaseName.Data.name}
    expect: root/${params.runId}
`), 0600))
	service := New(endly.New)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "root.yaml"), Format: "yaml", Content: `pipeline:
  child:
    action: run
    sharedState: true
    request: '@child'
`})
	require.NoError(t, err)
	for _, name := range []string{"first", "second"} {
		op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root", Params: map[string]interface{}{"runId": name}})
		require.NoError(t, err)
		op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		if op.Status != OperationSucceeded {
			for _, event := range op.Events {
				if strings.Contains(event.Type, "validator") || strings.Contains(event.Type, "Stdout") || strings.Contains(event.Type, "Stdin") {
					raw, _ := json.Marshal(event)
					t.Logf("%s: %s", name, raw)
				}
			}
		}
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
		require.Zero(t, op.FailedAssertions)
	}
}
