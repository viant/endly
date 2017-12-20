package endly

import (
	"fmt"
	"strings"
)

//EvaluateCriteria evaluates passed in criteria, it uses  <actual>:<expected>
//Assertion expression can be used for more complex criteria evaluation
func EvaluateCriteria(context *Context, criteria, eventType string, defaultValue bool) (bool, error) {
	if criteria == "" {
		return defaultValue, nil
	}
	colonPosition := strings.LastIndex(criteria, ":")
	if colonPosition == -1 {
		return false, fmt.Errorf("eval criteria needs to have colon: but had: %v", criteria)
	}
	fragments := []string{
		string(criteria[:colonPosition]),
		string(criteria[colonPosition+1:]),
	}
	var state = context.state
	actualOperand := state.Expand(strings.TrimSpace(fragments[0]))
	expectedOperand := state.Expand(strings.TrimSpace(fragments[1]))
	validator := &Validator{}
	var result, err = validator.Check(expectedOperand, actualOperand)
	AddEvent(context, eventType, Pairs("defaultValue", defaultValue, "actual", actualOperand, "expected", expectedOperand, "eligible", result), Info)
	return result, err
}
