package rest

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/endly/msg"
)

//Messages returns messages
func (r *Request) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyledText(fmt.Sprintf("%v %v", r.Method, r.URL), msg.MessageStyleGeneric), msg.NewStyledText("rest.Request", msg.MessageStyleGeneric)))
	requestJSON, _ := toolbox.AsJSONText(r)
	response = append(response, msg.NewMessage(msg.NewStyledText("Request", msg.MessageStyleGeneric), msg.NewStyledText("rest.Request", msg.MessageStyleGeneric),
		msg.NewStyledText(requestJSON, msg.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	responseJSON, _ := toolbox.AsJSONText(r)
	response = append(response, msg.NewMessage(msg.NewStyledText("Response", msg.MessageStyleGeneric), msg.NewStyledText("rest.Response", msg.MessageStyleGeneric),
		msg.NewStyledText(responseJSON, msg.MessageStyleOutput),
	))
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
