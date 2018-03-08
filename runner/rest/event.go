package reset

import (
	"github.com/viant/endly"
	"fmt"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *Request) Messages() []*endly.Message {
	var response = make([]*endly.Message, 0)
	response = append(response, endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v %v", r.Method, r.URL), endly.MessageStyleGeneric), endly.NewStyledText("rest.Request", endly.MessageStyleGeneric)))
	requestJSON, _ := toolbox.AsJSONText(r)
	response = append(response, endly.NewMessage(endly.NewStyledText("Request", endly.MessageStyleGeneric), endly.NewStyledText("rest.Request", endly.MessageStyleGeneric),
		endly.NewStyledText(requestJSON, endly.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*endly.Message {
	var response = make([]*endly.Message, 0)
	responseJSON, _ := toolbox.AsJSONText(r)
	response = append(response, endly.NewMessage(endly.NewStyledText("Response", endly.MessageStyleGeneric), endly.NewStyledText("rest.Response", endly.MessageStyleGeneric),
		endly.NewStyledText(responseJSON, endly.MessageStyleOutput),
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



