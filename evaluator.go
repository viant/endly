package endly

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly/criteria"
	"github.com/viant/toolbox/data"
)

//CriteriaEvalEvent represents criteria event
type CriteriaEvalEvent struct {
	Type             string
	Default          bool
	Evaluation       bool
	Criteria         string
	ExpandedCriteria string
	Error            string
}

//NewCriteriaEvalEvent creates a new evaluation event.
func NewCriteriaEvalEvent(criteriaType string, defaultValue, evaluation bool, criteria, expendedCriteria string, err error) *CriteriaEvalEvent {
	var result = &CriteriaEvalEvent{
		Type:             criteriaType,
		Default:          defaultValue,
		Evaluation:       evaluation,
		Criteria:         criteria,
		ExpandedCriteria: expendedCriteria,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	}
	return result
}

//Evaluate evaluates passed in criteria
func Evaluate(context *Context, state data.Map, criteriaExpression, eventType string, defaultValue bool) (bool, error) {
	if criteriaExpression == "" {
		return defaultValue, nil
	}
	parser := criteria.NewParser()
	criteria, err := parser.Parse(criteriaExpression)
	if err != nil {
		return !defaultValue, fmt.Errorf("%v, %v", err, criteriaExpression)
	}
	result, err := criteria.IsTrue(state)
	expandedCriteria := state.Expand(criteriaExpression)
	LogF("[%v] -> %v", expandedCriteria, result)
	if context != nil {
		context.Publish(NewCriteriaEvalEvent(eventType, defaultValue, result, criteriaExpression, fmt.Sprintf("%s", expandedCriteria), err))
	}
	return result, err
}

//Assert validates expected against actual
func Assert(context *Context, root string, expected, actual interface{}) (*assertly.Validation, error) {
	ctx := assertly.NewDefaultContext()
	ctx.Context = context.Context
	var rootPath = assertly.NewDataPath(root)
	return assertly.AssertWithContext(expected, actual, rootPath, ctx)
}
