package http

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

//Messages returns messages
func (r *Request) Messages() []*endly.Message {
	var response = make([]*endly.Message, 0)
	response = append(response, endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v %v", r.Method, r.URL), endly.MessageStyleGeneric), endly.NewStyledText("http.Request", endly.MessageStyleGeneric)))

	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, endly.NewMessage(endly.NewStyledText("Headers", endly.MessageStyleGeneric),
			endly.NewStyledText("http.Request", endly.MessageStyleGeneric),
			endly.NewStyledText(value, endly.MessageStyleInput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, endly.NewMessage(endly.NewStyledText("Cookies", endly.MessageStyleGeneric),
			endly.NewStyledText("http.Request", endly.MessageStyleGeneric),
			endly.NewStyledText(value, endly.MessageStyleInput),
		))
	}
	response = append(response, endly.NewMessage(endly.NewStyledText("Body", endly.MessageStyleGeneric), endly.NewStyledText("http.Request", endly.MessageStyleGeneric),
		endly.NewStyledText(r.Body, endly.MessageStyleInput),
	))
	return response
}

//Messages returns messages
func (r *Response) Messages() []*endly.Message {
	var response = make([]*endly.Message, 0)
	response = append(response, endly.NewMessage(endly.NewStyledText(fmt.Sprintf("StatusCode: %v", r.Code), endly.MessageStyleGeneric), endly.NewStyledText("http.Response", endly.MessageStyleGeneric)))
	if len(r.Header) > 0 {
		value, _ := toolbox.AsJSONText(r.Header)
		response = append(response, endly.NewMessage(endly.NewStyledText("Headers", endly.MessageStyleGeneric),
			endly.NewStyledText("http.Response", endly.MessageStyleGeneric),
			endly.NewStyledText(value, endly.MessageStyleOutput),
		))
	}
	if len(r.Cookies) > 0 {
		value, _ := toolbox.AsJSONText(r.Cookies)
		response = append(response, endly.NewMessage(endly.NewStyledText("Cookies", endly.MessageStyleGeneric),
			endly.NewStyledText("http.Response", endly.MessageStyleGeneric),
			endly.NewStyledText(value, endly.MessageStyleOutput),
		))
	}
	if r.Body != "" {
		response = append(response, endly.NewMessage(endly.NewStyledText("Body", endly.MessageStyleGeneric), endly.NewStyledText("http.Response", endly.MessageStyleGeneric),
			endly.NewStyledText(r.Body, endly.MessageStyleOutput),
		))
	}
	return response
}
