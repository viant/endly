package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func Test_EvaluateCriteria(t *testing.T) {

	manager := endly.NewManager()
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
			Description:   "Empty expression with default false",
			Expression:    "",
			DefaultResult: false,
			Expected:      false,
		},
	}

	for _, useCase := range useCases {

		if len(useCase.State) > 0 {
			for k, v := range useCase.State {
				state.Put(k, v)
			}
		}
		isTrue, err := endly.Evaluate(context, context.State(), useCase.Expression, "test", useCase.DefaultResult)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}

		assert.EqualValues(t, useCase.Expected, isTrue, useCase.Description)

	}

}
