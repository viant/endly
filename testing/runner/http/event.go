package http

import (
	"fmt"
	"github.com/viant/endly/msg"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *Request) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyled(fmt.Sprintf("%v %v", r.Method, r.URL), msg.MessageStyleGeneric), msg.NewStyled("http.ServiceRequest", msg.MessageStyleGeneric)))

	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, msg.NewMessage(msg.NewStyled("Headers", msg.MessageStyleGeneric),
			msg.NewStyled("http.ServiceRequest", msg.MessageStyleGeneric),
			msg.NewStyled(value, msg.MessageStyleInput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, msg.NewMessage(msg.NewStyled("Cookies", msg.MessageStyleGeneric),
			msg.NewStyled("http.ServiceRequest", msg.MessageStyleGeneric),
			msg.NewStyled(value, msg.MessageStyleInput),
		))
	}
	response = append(response, msg.NewMessage(msg.NewStyled("Body", msg.MessageStyleGeneric), msg.NewStyled("http.ServiceRequest", msg.MessageStyleGeneric),
		msg.NewStyled(r.Body, msg.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyled(fmt.Sprintf("StatusCode: %v", r.Code), msg.MessageStyleGeneric), msg.NewStyled("http.Response", msg.MessageStyleGeneric)))
	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, msg.NewMessage(msg.NewStyled("Headers", msg.MessageStyleGeneric),
			msg.NewStyled("http.Response", msg.MessageStyleGeneric),
			msg.NewStyled(value, msg.MessageStyleOutput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, msg.NewMessage(msg.NewStyled("Cookies", msg.MessageStyleGeneric),
			msg.NewStyled("http.Response", msg.MessageStyleGeneric),
			msg.NewStyled(value, msg.MessageStyleOutput),
		))
	}
	if r.Body != "" {
		response = append(response, msg.NewMessage(msg.NewStyled("Body", msg.MessageStyleGeneric), msg.NewStyled("http.Response", msg.MessageStyleGeneric),
			msg.NewStyled(r.Body, msg.MessageStyleOutput),
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
