package model

import (
	"testing"
	"github.com/viant/toolbox/data"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"io/ioutil"
)




func TestVariable_Apply(t *testing.T) {

	var useCases = []struct {
		Description string
		Variable    *Variable
		Input       map[string]interface{}
		Expected    map[string]interface{}
		HasError    bool
	}{
		{
			Description: "From assigment",
			Variable:    NewVariable("var1", "var2", "", false, nil, nil, nil),
			Input: map[string]interface{}{
				"var2": 123,
			},
			Expected: map[string]interface{}{
				"var1": 123,
			},
		},
		{
			Description: "Value ref assigment",
			Variable:    NewVariable("var1", "", "", false, "$var2", nil, nil),
			Input: map[string]interface{}{
				"var2": 123,
			},
			Expected: map[string]interface{}{
				"var1": 123,
			},
		},
		{
			Description: "From ref assigment",
			Variable:    NewVariable("var1", "$ref", "", false, "", nil, nil),
			Input: map[string]interface{}{
				"var2": 123,
				"ref":  "var2",
			},
			Expected: map[string]interface{}{
				"var1": 123,
			},
		},
		{
			Description: "Required value",
			Variable:    NewVariable("var1", "var2", "", true, "", nil, nil),
			Input: map[string]interface{}{
				"ref": "var2",
			},
			HasError: true,
		},
		{
			Description: "Conditional value assignment",
			Variable:    NewVariable("var1", "", "${in.var2} =< 10", false, "$var3", "$var4", nil),
			Input: map[string]interface{}{
				"var2": 10,
				"var3": 20,
				"var4": 30,
			},
			Expected: map[string]interface{}{
				"var1": 20,
			},
		},
		{
			Description: "Conditional else assignment",
			Variable:    NewVariable("var1", "", "${in.var2} > 10", false, "$var3", "$var4", nil),
			Input: map[string]interface{}{
				"var2": 10,
				"var3": 20,
				"var4": 30,
			},
			Expected: map[string]interface{}{
				"var1": 30,
			},
		},
		{
			Description: "Required fallback",
			Variable:    NewVariable("var1", "var2", "", false, "$var3", "", nil),
			Input: map[string]interface{}{
				"var3": 30,
			},
			Expected: map[string]interface{}{
				"var1": 30,
			},
		},

		{
			Description: "Post increment",
			Variable:    NewVariable("var1", "", "", false, "$var2++", "", nil),
			Input: map[string]interface{}{
				"var2": 10,
			},
			Expected: map[string]interface{}{
				"var1": 10,
			},
		},
		{
			Description: "Pre increment",
			Variable:    NewVariable("var1", "", "", false, "$++var2", "", nil),
			Input: map[string]interface{}{
				"var2": 10,
			},
			Expected: map[string]interface{}{
				"var1": 11,
			},
		},
		{
			Description: "Push",
			Variable:    NewVariable("->var1", "", "", false, "$var2", "", nil),
			Input: map[string]interface{}{
				"var2": 12,
			},
			Expected: map[string]interface{}{
				"var1": []int{12},
			},
		},

		{
			Description: "Unshift",
			Variable:    NewVariable("var1", "", "", false, "$<-var2", "", nil),
			Input: map[string]interface{}{
				"var2": []int{11, 12},
			},
			Expected: map[string]interface{}{
				"var1": 11,
			},
		},

		{
			Description: "Replace text value",
			Variable: NewVariable("var1", "", "", false, "$var2", "", map[string]string{
				"my": "endly",
			}),
			Input: map[string]interface{}{
				"var2": "this is my test",
			},
			Expected: map[string]interface{}{
				"var1": "this is endly test",
			},
		},

		{
			Description: "Replace skip non string value",
			Variable: NewVariable("var1", "", "", false, "$var2", "", map[string]string{
				"my": "endly",
			}),
			Input: map[string]interface{}{
				"var2": 123,
			},
			Expected: map[string]interface{}{
				"var1": 123,
			},
		},
	}
	for _, useCase := range useCases {
		var input = data.Map(useCase.Input)
		var output = data.NewMap()
		err := useCase.Variable.Apply(input, output)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if ! assert.Nil(t, err, useCase.Description) {
			continue
		}
		assertly.AssertValues(t, useCase.Expected, output, useCase.Description)
	}

}

func TestVariable_PersistValue(t *testing.T) {

	var var1 = NewVariable("key1", "", "", false, "123", nil, nil)
	toolbox.RemoveFileIfExist(var1.tempfile())
	var1.PersistValue()

	{ //load persisted value
		var2 := NewVariable("key1", "", "", false, "123", nil, nil)
		err := var2.Load()
		assert.Nil(t, err)
		assert.EqualValues(t, "123", var2.Value)

	}
	{
		toolbox.RemoveFileIfExist(var1.tempfile())
		var2 := NewVariable("key1", "", "", false, nil, nil, nil)
		toolbox.RemoveFileIfExist(var1.tempfile())
		ioutil.WriteFile(var1.tempfile(), []byte("aa"), 0644)
		err := var2.Load()
		assert.NotNil(t, err)
	}
	{
		toolbox.RemoveFileIfExist(var1.tempfile())
		var2 := NewVariable("key1", "", "", false, nil, nil, nil)
		toolbox.RemoveFileIfExist(var1.tempfile())
		err := var2.Load()
		assert.Nil(t, err)
		assert.Nil(t, var2.Value)
	}
}

func TestVariables_Apply(t *testing.T) {
	var variables Variables = []*Variable{
		NewVariable("var1", "var2", "", false, nil, nil, nil),
		nil,
		NewVariable("var4", "var3", "", false, nil, nil, nil),
	}

	var input = data.Map(map[string]interface{}{
		"var2": 123,
		"var3": 234,
	})

	{
		var output= data.NewMap()
		err := variables.Apply(input, output)
		assert.Nil(t, err)
		assert.EqualValues(t, 123, output.GetInt("var1"))
		assert.EqualValues(t, 234, output.GetInt("var4"))
	}
	{
		var output= data.NewMap()
		err := variables.Apply(nil, output)
		assert.Nil(t, err)
		assert.EqualValues(t, nil, output.Get("var1"))
		assert.EqualValues(t, nil, output.Get("var4"))
	}

	assert.NotNil(t, 	variables.Apply(nil, nil))
}

func TestVariables_String(t *testing.T) {
	var variables Variables = []*Variable{
		NewVariable("var1", "var2", "", false, nil, nil, nil),
		nil,
		NewVariable("var4", "var3", "", false, nil, nil, nil),
	}
	assert.EqualValues(t, `{Name:var1 From:var2 Value:<nil>},{Name:var4 From:var3 Value:<nil>},`, variables.String())
}

func TestVariableExpression_AsVariable(t *testing.T) {
	var useCases = []struct {
		Description string
		Expression  string
		Expected    *Variable
		HasError    bool
	}{
		{
			Description: "simple assignment",
			Expression:  "var1 = 123",
			Expected:    NewVariable("var1", "", "", false, "123", nil, nil),
		},
		{
			Description: "required simple assignment",
			Expression:  "! var1 = 123",
			Expected:    NewVariable("var1", "", "", true, "123", nil, nil),
		},
		{
			Description: "quoted assignment",
			Expression:  "var1 = '123 56'",
			Expected:    NewVariable("var1", "", "", false, "123 56", nil, nil),
		},
		{
			Description: "data structure assignment",
			Expression:  "var1 = [1, 2, 3]",
			Expected:    NewVariable("var1", "", "", false, []interface{}{1.0, 2.0, 3.0}, nil, nil),
		},

		{
			Description: "conditional assignment",
			Expression:  "$in.var2 > 10 ? var1 = [1, 2, 3]",
			Expected:    NewVariable("var1", "", "$in.var2 > 10 ", false, []interface{}{1.0, 2.0, 3.0}, nil, nil),
		},
		{
			Description: "conditional assignment with else",
			Expression:  "$in.var2 > 10 ? var1 = [1, 2, 3]:3",
			Expected:    NewVariable("var1", "", "$in.var2 > 10 ", false, []interface{}{1.0, 2.0, 3.0}, "3", nil),
		},
		{
			Description: "error assignment ",
			Expression:  "avc",
			HasError:true,
		},
	}

	for _, useCase := range useCases {
		var expr = VariableExpression(useCase.Expression)
		variable, err := expr.AsVariable()
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if !  assert.Nil(t, err, useCase.Description) {
			continue
		}
		assert.EqualValues(t, useCase.Expected, variable, useCase.Description)
	}
}

func TestGetVariables(t *testing.T) {


	var variables Variables = []*Variable{
		NewVariable("var1", "", "", false, "123", nil, nil),
	}

	{//variables use case
		actual, err := GetVariables("", variables)
		assert.Nil(t, err)
		assert.EqualValues(t, variables, actual)
	}

	{//*variables use case
		actual, err := GetVariables("", &variables)
		assert.Nil(t, err)
		assert.EqualValues(t, variables, actual)
	}

	{//expression use case
		actual, err := GetVariables("", []string{
			"var1 = 123",
		})
		assert.Nil(t, err)
		assert.EqualValues(t, variables, actual)
	}

	{//slice of map items use case
		actual, err := GetVariables("", []interface{}{
			map[string]interface{}{
				"Name":"var1",
				"Value":"123",
			},
		})
		assert.Nil(t, err)
		assert.EqualValues(t, variables, actual)
	}
	{//load from file use case
		var JSON = `[{"Name":"var1", "Value":"123"}]`
		ioutil.WriteFile("/tmp/endly_model_get_variables.json", []byte(JSON), 0644)
		actual, err := GetVariables("", "@/tmp/endly_model_get_variables.json")
		assert.Nil(t, err)
		assert.EqualValues(t, variables, actual)

	}
	{//no varibles
		actual, err := GetVariables("", "")
		assert.Nil(t, err)
		assert.Nil(t, actual)
	}

	{//no such file error case
		_, err := GetVariables("", "@nonexisting")
		assert.NotNil(t, err)
	}

	{//invalid expression error case
		_, err := GetVariables("", []string{"ac"})
		assert.NotNil(t, err)
	}

}