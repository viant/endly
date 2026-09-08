package webdriver

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
	"strings"
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
	for _, navigation := range r.Navigations {
		if navigation == nil {
			continue
		}
		result = append(result, msg.NewMessage(
			msg.NewStyled("Navigation", msg.MessageStyleGeneric),
			msg.NewStyled(navigation.StopReason, msg.MessageStyleOutput),
			msg.NewStyled(fmt.Sprintf("%s; target=%s; %dms; %d steps; height %d→%dpx; timedOut=%t; loadingStopped=%t; warning=%s", navigation.URL, navigation.ScrollTarget, navigation.ElapsedMs, navigation.Steps, navigation.StartHeightPx, navigation.FinalHeightPx, navigation.TimedOut, navigation.LoadingStopped, navigation.Warning), msg.MessageStyleOutput),
		))
	}
	for _, failure := range r.Failures {
		if failure == nil {
			continue
		}
		locations := make([]string, 0, 3)
		for _, location := range []string{failure.ScreenshotURL, failure.PageSourceURL, failure.MetadataURL} {
			if location != "" {
				locations = append(locations, location)
			}
		}
		result = append(result, msg.NewMessage(
			msg.NewStyled("Failure evidence", msg.MessageStyleError),
			msg.NewStyled(failure.Method, msg.MessageStyleOutput),
			msg.NewStyled(strings.Join(locations, ", "), msg.MessageStyleOutput),
		))
	}
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
