package model

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
)

//SliceKey represents slice key
const SliceKey = "data"

//Repeater represent repeated execution
type Repeater struct {
	Extract     Extracts  //textual regexp based data extraction
	Variables   Variables //structure data based data extraction
	Repeat      int       //how many time send this request
	SleepTimeMs int       //Sleep time after request send, this only makes sense with repeat option
	Exit        string    //Exit criteria, it uses expected variable to determine repeat termination
}

//Get returns non empty instance of default instance
func (r *Repeater) Init() *Repeater {
	if r == nil {
		repeater := NewRepeater()
		r = repeater
	}
	if r.Repeat == 0 {
		r.Repeat = 1
	}
	return r
}

//EvaluateExitCriteria check is exit criteria is met.
func (r *Repeater) EvaluateExitCriteria(callerInfo string, context *endly.Context, extracted map[string]interface{}) (bool, error) {
	var state = context.State()
	var extractedState = state.Clone()
	for k, v := range extracted {
		extractedState[k] = v
	}
	canBreak, err := criteria.Evaluate(context, extractedState, r.Exit, callerInfo, false)
	if err != nil {
		return true, fmt.Errorf("failed to check %v exit criteia: %v", callerInfo, err)
	}
	if canBreak {
		return true, nil
	}
	return false, nil

}

func (r *Repeater) runOnce(service *endly.AbstractService, callerInfo string, context *endly.Context, handler func() (interface{}, error), extracted map[string]interface{}) (bool, error) {
	out, err := handler()
	if err != nil {
		return false, err
	}
	if out == nil {
		return true, nil
	}
	extractableOutput, structuredOutput := util.AsExtractable(out)
	if len(structuredOutput) > 0 {
		if len(r.Variables) > 0 {
			err = r.Variables.Apply(structuredOutput, extracted)
		}
		if extractableOutput == "" {
			extractableOutput, _ = toolbox.AsJSONText(structuredOutput)
		}
	} else {
		err = r.Variables.Apply(extracted, extracted)
	}

	err = r.Extract.Extract(context, extracted, extractableOutput)
	if err != nil {
		return false, err
	}
	if extractableOutput != "" {
		extracted["output"] = extractableOutput //string output is published as $value
	}
	if r.Exit != "" {
		context.Publish(NewExtractEvent(extractableOutput, structuredOutput, extracted))
		if shouldBreak, err := r.EvaluateExitCriteria(callerInfo+"ExitEvaluation", context, extracted); shouldBreak || err != nil {
			return !shouldBreak, err
		}
	}
	return true, nil
}

//Run repeats x times supplied handler
func (r *Repeater) Run(service *endly.AbstractService, callerInfo string, context *endly.Context, handler func() (interface{}, error), extracted map[string]interface{}) error {
	for i := 0; i < r.Repeat; i++ {
		shouldContinue, err := r.runOnce(service, callerInfo, context, handler, extracted)
		if err != nil || !shouldContinue {
			return err
		}
		service.Sleep(context, r.SleepTimeMs)
	}
	return nil
}

//NewRepeater creates a new repeatable struct
func NewRepeater() *Repeater {
	return &Repeater{
		Repeat: 1,
	}
}
