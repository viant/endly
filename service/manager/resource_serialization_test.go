package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/internal/udf"
	_ "github.com/viant/endly/service/system/storage"
	_ "github.com/viant/endly/service/testing/validator"
)

func TestNativeResourceSerializationUploadReadback(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "fixture.yaml"), []byte(`- name: fixture
  value:
    id: $symbols.demo.ID
    count: 0
    enabled: false
    nullable: null
    emptyList: []
    emptyMap: {}
    literal.key[0]: unchanged
    rows:
    - id: $symbols.demo.ID
      amount: 1.25
`), 0600))
	service := New(endly.New)
	defer service.Shutdown(context.Background())
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	output := filepath.Join(root, "oracle.json")
	workflow := fmt.Sprintf(`init:
- name: symbols
  value: {demo: {ID: 9007199254740993}}
pipeline:
  native:
    action: nop
    init: '@fixture'
  upload:
    action: storage:upload
    init:
    - name: serialized
      value: $StateJSONBytes('fixture')
    sourceKey: serialized
    dest: {URL: '%s'}
  readback:
    action: validator:assert
    init:
    - name: decoded
      value: $LoadJSON('%s')
    actual: decoded
    expect:
      id: 9007199254740993
      count: 0
      enabled: false
      nullable: null
      emptyList: []
      emptyMap: {}
      literal.key[0]: unchanged
      rows:
      - id: 9007199254740993
        amount: 1.25
`, output, output)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "run.yaml"), Format: "yaml", Content: workflow})
	require.NoError(t, err)
	op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root"})
	require.NoError(t, err)
	op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationSucceeded, op.Status, op.Error)
	require.Zero(t, op.FailedAssertions)
	require.Positive(t, op.PassedAssertions, "native readback assertions must execute")
	encoded, err := os.ReadFile(output)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":9007199254740993,"count":0,"enabled":false,"nullable":null,"emptyList":[],"emptyMap":{},"literal.key[0]":"unchanged","rows":[{"id":9007199254740993,"amount":1.25}]}`, string(encoded))
}
