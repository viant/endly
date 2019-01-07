package rest

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *Request) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	response = append(response, msg.NewMessage(msg.NewStyled(fmt.Sprintf("%v %v", r.Method, r.URL), msg.MessageStyleGeneric), msg.NewStyled("rest.Request", msg.MessageStyleGeneric)))
	requestJSON, _ := toolbox.AsJSONText(r.Request)
	response = append(response, msg.NewMessage(msg.NewStyled("Request", msg.MessageStyleGeneric), msg.NewStyled("rest.Request", msg.MessageStyleGeneric),
		msg.NewStyled(requestJSON, msg.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*msg.Message {
	var response = make([]*msg.Message, 0)
	responseJSON, _ := toolbox.AsJSONText(r)
	response = append(response, msg.NewMessage(msg.NewStyled("Response", msg.MessageStyleGeneric), msg.NewStyled("rest.Response", msg.MessageStyleGeneric),
		msg.NewStyled(responseJSON, msg.MessageStyleOutput),
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
