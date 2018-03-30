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
		msg.NewMessage(msg.NewStyled("Response", msg.MessageStyleGeneric), msg.NewStyled("selenium", msg.MessageStyleGeneric),
			msg.NewStyled(seleniumData, msg.MessageStyleInput),
		))

	for _, errMessage := range r.LookupErrors {
		result = append(result,
			msg.NewMessage(msg.NewStyled(errMessage, msg.MessageStyleError), msg.NewStyled("selenium", msg.MessageStyleGeneric)))
	}

	for k, v := range r.Data {
		value := v
		if toolbox.IsCompleteJSON(toolbox.AsString(v)) {
			value, _ = toolbox.AsJSONText(v)
		}
		var message=fmt.Sprint("$Data.%v = %v", k, value)
		result = append(result,
			msg.NewMessage(msg.NewStyled(message, msg.MessageStyleOutput), msg.NewStyled("selenium.", msg.MessageStyleGeneric)))
	}
	return result
}

//IsInput returns this request (CLI reporter interface)
func (r *RunRequest) Messages() []*msg.Message {
	var result = make([]*msg.Message, 0)

	var actionCalls = make([]*msg.Styled, 0)
	for _, action := range r.Actions {
		var selector = ""
		if action.Selector != nil {
			selector = fmt.Sprintf("(%v:%v)", action.Selector.By, action.Selector.Value)
		}
		for _, call := range action.Calls {
			actionCalls = append(actionCalls, msg.NewStyled(fmt.Sprintf("%v.%v(%v)", selector, call.Method, call.Parameters), msg.MessageStyleGeneric))
		}
	}
	result = append(result,
		msg.NewMessage(msg.NewStyled("Run", msg.MessageStyleInput), msg.NewStyled("selenium.run", msg.MessageStyleGeneric),
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
