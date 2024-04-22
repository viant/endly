package webdriver

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
)

// Messages returns messages
func (r *RunResponse) Messages() []*msg.Message {
	var result = make([]*msg.Message, 0)

	var dataMessages = []*msg.Styled{}
	for k, v := range r.Data {
		value := v
		if toolbox.IsStructuredJSON(toolbox.AsString(v)) {
			value, _ = toolbox.AsJSONText(v)
		}
		dataMessages = append(dataMessages, msg.NewStyled(fmt.Sprintf("%v = %v", k, value), msg.MessageStyleOutput))
	}
	result = append(result,
		msg.NewMessage(msg.NewStyled("Response", msg.MessageStyleGeneric), msg.NewStyled("selenium", msg.MessageStyleGeneric), dataMessages...))
	if len(r.LookupErrors) == 0 {
		return result
	}
	for _, errMessage := range r.LookupErrors {
		result = append(result,
			msg.NewMessage(msg.NewStyled(errMessage, msg.MessageStyleOutput), msg.NewStyled("lookup", msg.MessageStyleError)))
	}
	return result
}

// IsInput returns this request (CLI reporter interface)
func (r *RunRequest) Messages() []*msg.Message {
	var result = make([]*msg.Message, 0)

	var actionCalls = make([]*msg.Styled, 0)
	for _, action := range r.Actions {
		var selector = ""
		if action.Selector != nil {
			selector = fmt.Sprintf("(%v:%v)", action.Selector.By, action.Selector.Value)
		}
		if selector != "" {
			selector += "."
		}
		for _, call := range action.Calls {
			actionCalls = append(actionCalls, msg.NewStyled(fmt.Sprintf("%v%v(%v)", selector, call.Method, call.Parameters), msg.MessageStyleInput))
		}
	}
	result = append(result,
		msg.NewMessage(msg.NewStyled("Request", msg.MessageStyleGeneric), msg.NewStyled("selenium.run", msg.MessageStyleGeneric),
			actionCalls...))

	return result
}

// IsInput returns this request (CLI reporter interface)
func (r *RunRequest) IsInput() bool {
	return true
}

// IsOutput returns this response (CLI reporter interface)
func (r *RunResponse) IsOutput() bool {
	return true
}
