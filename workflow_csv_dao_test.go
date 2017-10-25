package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
)

func TestNewWorkflowDao(t *testing.T) {

	{
		endly.UdfRegistry["udf1"] = func(source interface{}, state data.Map) (interface{}, error) {
			text := toolbox.AsString(source)
			return strings.ToUpper(text), nil
		}
		dao := endly.NewWorkflowDao()
		conext := &endly.Context{Context: toolbox.NewContext()}
		workflow, err := dao.Load(conext, url.NewResource("test/workflow/w1.csv"))

		if assert.Nil(t, err) {
			if assert.NotNil(t, workflow) {
				assert.Equal(t, "Simple Test", workflow.Name)
				assert.Equal(t, "My description", workflow.Description)

				assert.Equal(t, 4, len(workflow.Tasks))
				assert.Equal(t, "Simple Http Test", workflow.Tasks[0].Name)
				assert.Equal(t, 1, len(workflow.Tasks[0].Init))
				assert.Equal(t, "v10", workflow.Tasks[0].Init[0].Name)
				assert.Equal(t, 2, len(workflow.Tasks[1].Init))

				assert.Equal(t, "v30", workflow.Tasks[2].Init[0].Name)

				assert.Equal(t, "v1", workflow.Data.GetString("k1"))
				assert.Equal(t, "v2", workflow.Data.GetString("k2"))
				assert.Equal(t, "v3", workflow.Data.GetString("k3"))

				if assert.True(t, workflow.Data.Has("Arr")) {
					var collection = toolbox.AsSlice(workflow.Data.GetCollection("Arr"))
					assert.Equal(t, []interface{}{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0}, collection)
				}
				assert.Equal(t, "ABC", workflow.Data.GetString("Udf"))
				//test expansion ^
				assert.Equal(t, "123", workflow.Tasks[3].Init[0].Value)
			}
		}
	}
	{
		dao := endly.NewWorkflowDao()
		conext := &endly.Context{Context: toolbox.NewContext()}
		workflow, err := dao.Load(conext, url.NewResource("test/workflow/simple.csv"))
		if assert.Nil(t, err) {
			assert.NotNil(t, workflow)
			task := workflow.Tasks[0]
			assert.Equal(t, "First Set of requestss", task.Name)
			assert.Equal(t, 2, len(task.Actions))

			action := task.Actions[0]
			assert.Equal(t, "send", action.Action)
			var request, ok = action.Request.(data.Map)
			assert.True(t, ok)
			assert.NotNil(t, request["Requests"])
			requests, ok := request["Requests"].(*data.Collection)
			assert.True(t, ok)

			assert.Equal(t, 3, len(*requests))
			httpReuest, ok := (*requests)[1].(data.Map)
			assert.True(t, ok)

			assert.Equal(t, "$appServer/path2", httpReuest.GetString("URL"))
			assert.Equal(t, "GET", httpReuest.GetString("Method"))
		}
	}
}
