package criteria

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria/compiler"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/toolbox/data"
)

// EvalEvent represents criteria event
type EvalEvent struct {
	Type       string
	Default    bool
	Evaluation bool
	Criteria   string
	Has        bool
	Error      string
}

// NewEvalEvent creates a new evaluation event.
func NewEvalEvent(criteriaType string, defaultValue, evaluation bool, criteria string, has bool, err error) *EvalEvent {
	var result = &EvalEvent{
		Type:       criteriaType,
		Default:    defaultValue,
		Evaluation: evaluation,
		Has:        has,
		Criteria:   criteria,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	}
	return result
}

func Evaluate(context *endly.Context, state data.Map, expression string, compute *eval.Compute, eventType string, defaultValue bool) (bool, error) {
	if expression == "" {
		return defaultValue, nil
	}
	var evaluator eval.Compute
	if *compute != nil {
		evaluator = *compute
	}
	if evaluator == nil {
		cmp, err := compiler.Compile(expression)
		if err != nil {
			return defaultValue, err
		}
		evaluator, err = cmp()
		if err != nil {
			return defaultValue, err
		}
	}
	result, has, err := evaluator(state)
	ret, ok := result.(bool)
	if !ok {
		ret = defaultValue
	}
	if context != nil {
		context.Publish(NewEvalEvent(eventType, defaultValue, ret, expression, true, err))
	}
	if err != nil {
		return defaultValue, err
	}
	if !has {
		return defaultValue, nil
	}
	return ret, nil
}

// Assert validates expected against actual
func Assert(context *endly.Context, root string, expected, actual interface{}) (*assertly.Validation, error) {
	ctx := assertly.NewDefaultContext()
	ctx.Context = context.Context
	var rootPath = assertly.NewDataPath(root)
	return assertly.AssertWithContext(expected, actual, rootPath, ctx)
}
