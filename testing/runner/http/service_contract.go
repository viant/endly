package http

import (
	"github.com/viant/endly"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//SendRequest represents a send http request.
type SendRequest struct {
	Options      []*toolbox.HttpOptions `description:"http client options: key value pairs, where key is one of the following: HTTP options:RequestTimeoutMs,TimeoutMs,KeepAliveTimeMs,TLSHandshakeTimeoutMs,ResponseHeaderTimeoutMs,MaxIdleConns"`
	Requests     []*Request
	Expect       interface{}          `description:"If specified it will validated response as actual"`
	UdfProviders []*endly.UdfProvider `description:"collection of predefined udf provider name with custom parameters and new registration id"`
}

//NewSendRequestFromURL create new request from URL
func NewSendRequestFromURL(URL string) (*SendRequest, error) {
	resource := url.NewResource(URL)
	var request = &SendRequest{}
	return request, resource.Decode(request)
}

//SendResponse represnets a send response
type SendResponse struct {
	Responses []*Response
	Data      data.Map
	Assert    *validator.AssertResponse
}

//NewResponse creates and appends a response
func (r *SendResponse) NewResponse() *Response {
	response := NewResponse()
	if len(r.Responses) == 0 {
		r.Responses = []*Response{}
	}
	r.Responses = append(r.Responses, response)
	return response
}

//Expands expands data ($httpTrips.Data) attribute shared across requests within a group
func (r *SendResponse) Expand(state data.Map) {
	if len(r.Data) == 0 {
		return
	}
	for i := 0; i < 3; i++ { //we expanding numerous time in case there are some multi level references
		//TODO add function to check if any unexpanded placeholder left
		expanded := r.Data.Expand(state)
		r.Data = data.Map(toolbox.AsMap(expanded))
	}
}

//NewSendRequestFromURL create new request from URL
func NewSendResponseFromURL(URL string) (*SendResponse, error) {
	resource := url.NewResource(URL)
	var request = &SendResponse{}
	return request, resource.Decode(request)
}
