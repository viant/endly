package criteria

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/toolbox"
	"testing"
)

func Test_EvaluateCriteria(t *testing.T) {

	manager := endly.New()

	var useCases = []struct {
		Description   string
		DefaultResult bool
		Expression    string
		State         map[string]interface{}
		Expected      bool
		HasError      bool
	}{

		{

			Expression: "${httpTrips.Response[0].Body}://auctionwon/",
			State: map[string]interface{}{
				"httpTrips": map[string]interface{}{
					"Response": []interface{}{
						map[string]interface{}{
							"Body": "http://auctionwon/",
						},
					},
				},
			},
			Expected: true,
		},

		{
			Description: "basic literal expr",
			Expression:  "on!=on", //
			State:       map[string]interface{}{},
			Expected:    false,
		},
		{
			Description: "basic literal expr",
			Expression:  "on=on", //
			State:       map[string]interface{}{},
			Expected:    true,
		},

		{
			Description:   "selector $key1:1",
			Expression:    "$checkAngular.Output:!/16.2.1/", //
			DefaultResult: true,
			Expected:      true,
			State: map[string]interface{}{
				"checkAngular.": map[string]interface{}{
					"Output": "3",
				},
			},
		},
		{
			Description:   "eq $key1:1",
			Expression:    "$key1:1", //
			DefaultResult: true,
			Expected:      true,
			State: map[string]interface{}{
				"key1": 1,
			},
		},
		{
			Description:   "ne $key1:!0",
			Expression:    "$key1:!0", //
			DefaultResult: true,
			Expected:      false,
			State: map[string]interface{}{
				"key1": 0,
			},
		},
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
			DefaultResult: false,
			Expected:      true,
		},
		{
			Description:   "Not equal 2",
			Expression:    "1!=0",
			DefaultResult: false,
			Expected:      true,
		},

		{
			Description:   "UDFs substitution",
			Expression:    "$Len($logRecords) > 0", //
			DefaultResult: true,
			Expected:      true,
			State: map[string]interface{}{
				"logRecords": []interface{}{"1"},
			},
		},
		{
			Description: "UDFs substitution",
			Expression:  "$Len($logRecords) > 2", //
			//DefaultResult: true,
			Expected: false,
			State: map[string]interface{}{
				"logRecords": []interface{}{"1"},
			},
		},
		{
			Description: "Uni operand expression",
			Expression:  "$getTrue()", //
			Expected:    true,
			State: map[string]interface{}{
				"getTrue": func() interface{} {
					return true
				},
			},
		},
		{
			Description: "contains",
			Expression:  "$bar() contains abc", //
			Expected:    true,
			State: map[string]interface{}{
				"bar": func() interface{} {
					return "abcd"
				},
			},
		},

		{
			Description: "contains //",
			Expression:  "$a:/abc/", //
			Expected:    true,
			State: map[string]interface{}{
				"a": "abcxv",
			},
		},

		{
			Description: "contains - not defiend",
			Expression:  "$bar() contains abc", //
			Expected:    false,
			State:       map[string]interface{}{},
		},
		{
			Description: "contains - //",
			Expression:  "$bar():/abc/", //
			Expected:    true,
			State: map[string]interface{}{
				"bar": func() interface{} {
					return "abcr"
				},
			},
		},
	}

	for i, useCase := range useCases {
		context := manager.NewContext(toolbox.NewContext())
		state := context.State()
		if len(useCase.State) > 0 {
			for k, v := range useCase.State {
				state.Put(k, v)
			}
		}
		var e eval.Compute
		isTrue, err := Evaluate(context, state, useCase.Expression, &e, "test", useCase.DefaultResult)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.Expected, isTrue, fmt.Sprintf("case %v: %s\n", i, useCase.Description))
	}

}
