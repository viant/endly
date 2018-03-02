package endly

import (
	"fmt"
	"github.com/viant/toolbox/data"
)

//EvaluateCriteria evaluates passed in criteria, it uses  <actual>:<expected>
//
//Assertion expression can be used for more complex criteria evaluation
func EvaluateCriteria(context *Context, state data.Map, criteriaExpression, eventType string, defaultValue bool) (bool, error) {
	if criteriaExpression == "" {
		return defaultValue, nil
	}
	parser := NewCriteriaParser()
	criteria, err := parser.Parse(criteriaExpression)
	if err != nil {
		return !defaultValue, fmt.Errorf("%v, %v", err, criteriaExpression)
	}
	result, err := criteria.IsTrue(context, state)
	expandedCriteria := state.Expand(criteriaExpression)
	AddEvent(context, eventType, Pairs("defaultValue", defaultValue, "eligible", result, "err", fmt.Sprintf("%v", err), "criteria", expandedCriteria), Info)
	return result, err
}
