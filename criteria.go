package endly

import (
	"strings"
)

//EvaluateCriteria evaluates passed in criteria, it uses  <actual>:<expected>
//
//Assertion expression can be used for more complex criteria evaluation
func EvaluateCriteria(context *Context, criteria, eventType string, defaultValue bool) (bool, error) {
	if criteria == "" {
		return defaultValue, nil
	}
	criteria = strings.TrimSpace(criteria)
	colonPosition := strings.LastIndex(criteria, ":")
	if colonPosition == -1 {
		if strings.HasPrefix(criteria, "!$") {
			criteria = string(criteria[1:]) + ":"
		} else {
			criteria = criteria + ":!"
		}
		colonPosition = strings.LastIndex(criteria, ":")
	}

	fragments := []string{
		string(criteria[:colonPosition]),
		string(criteria[colonPosition+1:]),
	}
	var state = context.state
	actualOperand := state.Expand(strings.TrimSpace(fragments[0]))
	expectedOperand := state.Expand(strings.TrimSpace(fragments[1]))
	validation, err := Assert(context, "/", expectedOperand, actualOperand)
	var result = validation.FailedCount == 0
	if err != nil {
		return false, err
	}
	AddEvent(context, eventType, Pairs("defaultValue", defaultValue, "actual", actualOperand, "expected", expectedOperand, "eligible", result,
		"criteria", criteria,
		"leftOperand", fragments[0],
		"rightOperand", fragments[1],
	), Info)
	return result, err
}
