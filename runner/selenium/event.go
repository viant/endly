package selenium

import (
	"fmt"
	"github.com/viant/endly/msg"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *RunResponse) Messages() []*msg.Message {
	var result = make([]*msg.Message, 0)
	if len(r.LookupErrors) == 0 {
		return result
	}

	seleniumData, _ := toolbox.AsJSONText(r.Data)
	result = append(result,
		msg.NewMessage(msg.NewStyledText("Response", msg.MessageStyleGeneric), msg.NewStyledText("selenium", msg.MessageStyleGeneric),
			msg.NewStyledText(seleniumData, msg.MessageStyleInput),
		))

	for _, errMessage := range r.LookupErrors {
		result = append(result,
			msg.NewMessage(msg.NewStyledText(errMessage, msg.MessageStyleError), msg.NewStyledText("selenium", msg.MessageStyleGeneric)))
	}
	return result
}

//IsInput returns this request (CLI reporter interface)
func (r *RunRequest) Messages() []*msg.Message {
	var result = make([]*msg.Message, 0)

	var actionCalls = make([]*msg.StyledText, 0)
	for _, action := range r.Actions {
		var selector = "webDriver"
		if action.Selector != nil {
			selector = fmt.Sprintf("webElement[%v %v]", action.Selector.By, action.Selector.Key+action.Selector.Value)
		}
		for _, call := range action.Calls {
			actionCalls = append(actionCalls, msg.NewStyledText(fmt.Sprintf("%v.%v(%v)", selector, call.Method, call.Parameters), msg.MessageStyleGeneric))
		}
	}
	result = append(result,
		msg.NewMessage(msg.NewStyledText("Run", msg.MessageStyleOutput), msg.NewStyledText("selenium.run", msg.MessageStyleGeneric),
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
