package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewFieldExpression(t *testing.T) {

	expr := endly.NewFieldExpression("Request.[]Expected.Headers")
	{
		assert.True(t, expr.HasSubPath)
		assert.True(t, expr.HasArrayComponent)
		assert.False(t, expr.IsArray)
		assert.Equal(t, "Request", expr.Field)
	}
	{
		expr = expr.Child
		assert.True(t, expr.HasSubPath)
		assert.True(t, expr.HasArrayComponent)
		assert.True(t, expr.IsArray)
		assert.Equal(t, "Expected", expr.Field)
	}
	{
		expr = expr.Child
		assert.False(t, expr.HasSubPath)
		assert.False(t, expr.HasArrayComponent)
		assert.False(t, expr.IsArray)
		assert.Equal(t, "Headers", expr.Field)
	}

	expr = endly.NewFieldExpression("[]Arr")
	{
		assert.False(t, expr.HasSubPath)
		assert.True(t, expr.HasArrayComponent)
		assert.True(t, expr.IsArray)
		assert.Equal(t, "Arr", expr.Field)
	}
}

func TestFieldExpression_Set(t *testing.T) {

	{
		var object = common.NewMap()
		field1 := endly.NewFieldExpression("Field1")
		field1.Set(123, object)
		assert.Equal(t, 123, object.GetInt("Field1"))

	}

	{
		var object = common.NewMap()
		field1 := endly.NewFieldExpression("Req.[]Array.H")

		field1.Set("v1H", object)
		field1.Set("v2H", object, 1)

		field2 := endly.NewFieldExpression("Req.[]Array.A")
		field2.Set("v1A", object)
		field2.Set("v2A", object, 1)

		field3 := endly.NewFieldExpression("Req.Field")

		field3.Set("v", object)

		assert.True(t, object.Has("Req"))
		var reqObject = object.GetMap("Req")
		assert.NotNil(t, reqObject)

		assert.Equal(t, "v", reqObject.GetString("Field"))
		assert.True(t, reqObject.Has("Array"))
		assert.True(t, reqObject.Has("Field"))

		array := reqObject.GetCollection("Array")
		assert.NotNil(t, array)
		assert.Equal(t, 2, len(*array))

		array.RangeMap(func(item common.Map, index int) (bool, error) {
			switch index {

			case 0:
				assert.Equal(t, "v1H", item.GetString("H"))
				assert.Equal(t, "v1A", item.GetString("A"))

			case 1:
				assert.Equal(t, "v2H", item.GetString("H"))
				assert.Equal(t, "v2A", item.GetString("A"))

			}
			return true, nil
		})

	}

}

func TestNewWorkflowDao(t *testing.T) {

	{
		dao := endly.NewWorkflowDao()
		conext := &endly.Context{Context: toolbox.NewContext()}
		workflow, err := dao.Load(conext, endly.NewFileResource("test/workflow/w1.csv"))
		assert.Nil(t, err)
		assert.NotNil(t, workflow)
		assert.Equal(t, "Simple Test", workflow.Name)
		assert.Equal(t, "My description", workflow.Description)

		assert.Equal(t, 3, len(workflow.Tasks))
		assert.Equal(t, "Simple Http Test", workflow.Tasks[0].Name)
		assert.Equal(t, 1, len(workflow.Tasks[0].Variables))
		assert.Equal(t, "v10", workflow.Tasks[0].Variables[0].Name)
		assert.Equal(t, 2, len(workflow.Tasks[1].Variables))

		assert.Equal(t, "v30", workflow.Tasks[2].Variables[0].Name)

		assert.Equal(t, "v1", workflow.Data.GetString("k1"))
		assert.Equal(t, "v2", workflow.Data.GetString("k2"))
		assert.Equal(t, "v3", workflow.Data.GetString("k3"))

		if assert.True(t, workflow.Data.Has("Arr")) {
			var collection= toolbox.AsSlice(workflow.Data.GetCollection("Arr"))
			assert.Equal(t, []interface{}{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0}, collection)
		}
	}
	{
		dao := endly.NewWorkflowDao()
		conext := &endly.Context{Context: toolbox.NewContext()}
		workflow, err := dao.Load(conext, endly.NewFileResource("test/workflow/simple.csv"))
		assert.Nil(t, err)
		assert.NotNil(t, workflow)
		task := workflow.Tasks[0]
		assert.Equal(t, "First Set of requestss", task.Name)
		assert.Equal(t, 2, len(task.Actions))

		action := task.Actions[0]
		assert.Equal(t, "send", action.Action)
		var request, ok = action.Request.(common.Map)
		assert.True(t, ok)
		assert.NotNil(t, request["Requests"])
		requests, ok := request["Requests"].(*common.Collection)
		assert.True(t, ok)

		assert.Equal(t, 3, len(*requests))
		httpReuest, ok := (*requests)[1].(common.Map)
		assert.True(t, ok)

		assert.Equal(t, "$appServer/path2", httpReuest.GetString("URL"))
		assert.Equal(t, "GET", httpReuest.GetString("Method"))

	}
}
