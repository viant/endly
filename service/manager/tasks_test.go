package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

func TestInstanceDiscoveryMatchesNestedExecution(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"cases/001_one", "other/001_two"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name), 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "child.yaml"), []byte(`pipeline:
  core:
    subPath: cases/${index}_*
    template:
      execute:
        action: nop
        init:
          coreRan: yes
  other:
    subPath: other/${index}_*
    template:
      execute:
        action: nop
        init:
          otherRan: yes
`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "run.yaml"), []byte(`pipeline:
  test:
    action: run
    sharedState: true
    request: '@child'
`), 0644))
	service := New(endly.New)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, URL: filepath.Join(root, "run.yaml")})
	require.NoError(t, err)
	tree, err := service.ListTasks(&ListTasksRequest{SessionID: session.SessionID, Workflow: "run", ExpandWorkflows: true})
	require.NoError(t, err)
	require.Len(t, tree.Tasks[0].Workflows, 1)
	child := tree.Tasks[0].Workflows[0]
	require.Len(t, child.Tasks[0].Instances, 1)
	tag := child.Tasks[0].Instances[0].TagID
	operation, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "run", Tasks: "test", SelectorMode: "path", TagIDs: tag})
	require.NoError(t, err)
	operation, err = service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, operation.Status, operation.Error)
	core, err := service.InspectContext(&InspectContextRequest{SessionID: session.SessionID, Path: "coreRan"})
	require.NoError(t, err)
	require.True(t, core.Found)
	other, err := service.InspectContext(&InspectContextRequest{SessionID: session.SessionID, Path: "otherRan"})
	require.NoError(t, err)
	require.False(t, other.Found)
	operation, err = service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "run", Tasks: "test", SelectorMode: "path", TagIDs: "does-not-exist"})
	require.NoError(t, err)
	operation, err = service.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, OperationFailed, operation.Status)
	require.Contains(t, operation.Error, "no actions matched")

}
