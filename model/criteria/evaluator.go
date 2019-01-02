package criteria

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

//EvalEvent represents criteria event
type EvalEvent struct {
	Type             string
	Default          bool
	Evaluation       bool
	Criteria         string
	ExpandedCriteria string
	Error            string
}

//NewEvalEvent creates a new evaluation event.
func NewEvalEvent(criteriaType string, defaultValue, evaluation bool, criteria, expendedCriteria string, err error) *EvalEvent {
	var result = &EvalEvent{
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
func Evaluate(context *endly.Context, state data.Map, expression, eventType string, defaultValue bool) (bool, error) {
	if expression == "" {
		return defaultValue, nil
	}
	parser := NewParser()
	predicate, err := parser.Parse(expression)
	if err != nil {
		return !defaultValue, fmt.Errorf("%v, %v", err, expression)
	}
	result, err := predicate.Apply(state)
	expandedCriteria := state.Expand(expression)
	if context != nil {
		context.Publish(NewEvalEvent(eventType, defaultValue, result, expression, fmt.Sprintf("%v", expandedCriteria), err))
	}
	return result, err
}

//Assert validates expected against actual
func Assert(context *endly.Context, root string, expected, actual interface{}) (*assertly.Validation, error) {
	ctx := assertly.NewDefaultContext()
	ctx.Context = context.Context
	var rootPath = assertly.NewDataPath(root)
	return assertly.AssertWithContext(expected, actual, rootPath, ctx)
}
