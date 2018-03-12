package selenium

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *RunResponse) Messages() []*endly.Message {
	var result = make([]*endly.Message, 0)
	if len(r.LookupErrors) == 0 {
		return result
	}

	seleniumData, _ := toolbox.AsJSONText(r.Data)
	result = append(result,
		endly.NewMessage(endly.NewStyledText("Response", endly.MessageStyleGeneric), endly.NewStyledText("selenium", endly.MessageStyleGeneric),
			endly.NewStyledText(seleniumData, endly.MessageStyleInput),
		))

	for _, errMessage := range r.LookupErrors {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(errMessage, endly.MessageStyleError), endly.NewStyledText("selenium", endly.MessageStyleGeneric)))
	}
	return result
}

//IsInput returns this request (CLI reporter interface)
func (r *RunRequest) Messages() []*endly.Message {
	var result = make([]*endly.Message, 0)

	var actionCalls = make([]*endly.StyledText, 0)
	for _, action := range r.Actions {
		var selector = "webDriver"
		if action.Selector != nil {
			selector = fmt.Sprintf("webElement[%v %v]", action.Selector.By, action.Selector.Key+action.Selector.Value)
		}
		for _, call := range action.Calls {
			actionCalls = append(actionCalls, endly.NewStyledText(fmt.Sprintf("%v.%v(%v)", selector, call.Method, call.Parameters), endly.MessageStyleGeneric))
		}
	}
	result = append(result,
		endly.NewMessage(endly.NewStyledText("Run", endly.MessageStyleOutput), endly.NewStyledText("selenium.run", endly.MessageStyleGeneric),
			actionCalls...))

	return result
}

//IsInput returns this request (CLI reporter interface)
func (r *RunRequest) IsInput() bool {
	return true
}

//IsOutput returns this response (CLI reporter interface)
func (r *RunResponse) IsOutput() bool {
	return true
}
