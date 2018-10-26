package criteria

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func Test_EvaluateCriteria(t *testing.T) {

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	state := context.State()

	var useCases = []struct {
		Description   string
		DefaultResult bool
		Expression    string
		State         map[string]interface{}
		Expected      bool
		HasError      bool
	}{
		{
			Description: "Simple true expression with ok = true",
			Expression:  "$ok:true",
			State: map[string]interface{}{
				"ok": true,
			},
			Expected: true,
		},
		{
			Description: "Left operand expression",
			Expression:  "$key1",
			State: map[string]interface{}{
				"key1": "123",
			},
			Expected: true,
		},
		{
			Description: "Simple true expression with ok = false",
			Expression:  "$ok:true",
			State: map[string]interface{}{
				"ok": false,
			},
			Expected: false,
		},
		{
			Description:   "Empty expression with default true",
			Expression:    "",
			DefaultResult: true,
			Expected:      true,
		},
		{
			Description:   "Constant equal",
			Expression:    "a=a",
			DefaultResult: true,
			Expected:      true,
		},
		{
			Description:   "Empty expression with default false",
			Expression:    "",
			DefaultResult: false,
			Expected:      false,
		},
		{
			Description:   "Not equal",
			Expression:    "1:!0",
			DefaultResult: true,
			Expected:      true,
		},
		{
			Description:   "Not equal 2",
			Expression:    "1!=0",
			DefaultResult: true,
			Expected:      true,
		},
		{
			Description:   "Data substitution",
			Expression:    "$key1:!0", //
			DefaultResult: true,
			Expected:      false,
			State: map[string]interface{}{
				"key1": 0,
			},
		},
	}

	for _, useCase := range useCases {
		if len(useCase.State) > 0 {
			for k, v := range useCase.State {
				state.Put(k, v)
			}
		}
		isTrue, err := Evaluate(context, state, useCase.Expression, "test", useCase.DefaultResult)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.Expected, isTrue, useCase.Description)
	}

}
