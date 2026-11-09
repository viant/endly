package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
)

func TestInlinedActionRequestPreservesEmptyObjectIdentity(t *testing.T) {
	for _, withState := range []bool{false, true} {
		t.Run(map[bool]string{false: "without state", true: "with state"}[withState], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "shape.yaml")
			require.NoError(t, os.WriteFile(path, []byte(`pipeline:
  contract:
    requests:
      action: http/runner:send
      requests:
        - Method: PATCH
          URL: http://unused.invalid/audience
          JSONBody:
            data:
              - frequencyCapping: {}
                roles: []
                nullable: null
                nested:
                  object: {}
                  list: []
                name: literal
`), 0600))
			source, err := loadInlineWorkflow(path)
			require.NoError(t, err)
			if withState {
				source.State = data.Map{"unrelated": "value"}
			}
			workflow, err := source.AsWorkflow("shape", t.TempDir())
			require.NoError(t, err)
			var actions []*Action
			var collect func(*TasksNode)
			collect = func(node *TasksNode) {
				if node == nil {
					return
				}
				for _, task := range node.Tasks {
					actions = append(actions, task.Actions...)
					collect(task.TasksNode)
				}
			}
			collect(workflow.TasksNode)
			require.Len(t, actions, 1)
			encoded, err := json.Marshal(actions[0].Request)
			require.NoError(t, err)
			var request map[string]interface{}
			require.NoError(t, json.Unmarshal(encoded, &request))
			requests := request["requests"].([]interface{})
			body := requests[0].(map[string]interface{})["JSONBody"].(map[string]interface{})
			row := body["data"].([]interface{})[0].(map[string]interface{})
			require.Equal(t, map[string]interface{}{}, row["frequencyCapping"])
			require.Equal(t, []interface{}{}, row["roles"])
			value, present := row["nullable"]
			require.True(t, present)
			require.Nil(t, value)
			require.NotContains(t, row, "omitted")
			require.Equal(t, map[string]interface{}{"object": map[string]interface{}{}, "list": []interface{}{}}, row["nested"])
		})
	}
}
