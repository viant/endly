package http

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/endly/msg"
)

//Messages returns messages
func (r *Request) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyledText(fmt.Sprintf("%v %v", r.Method, r.URL), msg.MessageStyleGeneric), msg.NewStyledText("http.ServiceRequest", msg.MessageStyleGeneric)))

	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, msg.NewMessage(msg.NewStyledText("Headers", msg.MessageStyleGeneric),
			msg.NewStyledText("http.ServiceRequest", msg.MessageStyleGeneric),
			msg.NewStyledText(value, msg.MessageStyleInput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, msg.NewMessage(msg.NewStyledText("Cookies", msg.MessageStyleGeneric),
			msg.NewStyledText("http.ServiceRequest", msg.MessageStyleGeneric),
			msg.NewStyledText(value, msg.MessageStyleInput),
		))
	}
	response = append(response, msg.NewMessage(msg.NewStyledText("Body", msg.MessageStyleGeneric), msg.NewStyledText("http.ServiceRequest", msg.MessageStyleGeneric),
		msg.NewStyledText(r.Body, msg.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyledText(fmt.Sprintf("StatusCode: %v", r.Code), msg.MessageStyleGeneric), msg.NewStyledText("http.Response", msg.MessageStyleGeneric)))
	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, msg.NewMessage(msg.NewStyledText("Headers", msg.MessageStyleGeneric),
			msg.NewStyledText("http.Response", msg.MessageStyleGeneric),
			msg.NewStyledText(value, msg.MessageStyleOutput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, msg.NewMessage(msg.NewStyledText("Cookies", msg.MessageStyleGeneric),
			msg.NewStyledText("http.Response", msg.MessageStyleGeneric),
			msg.NewStyledText(value, msg.MessageStyleOutput),
		))
	}
	if r.Body != "" {
		response = append(response, msg.NewMessage(msg.NewStyledText("Body", msg.MessageStyleGeneric), msg.NewStyledText("http.Response", msg.MessageStyleGeneric),
			msg.NewStyledText(r.Body, msg.MessageStyleOutput),
		))
	}
	return response
}

//IsInput returns this request (CLI reporter interface)
func (r *Request) IsInput() bool {
	return true
}

//IsOutput returns this response (CLI reporter interface)
func (r *Response) IsOutput() bool {
	return true
}
