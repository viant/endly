package http

import (
	"fmt"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//SendRequest represents a send http request.
type SendRequest struct {
	Options     map[string]interface{} `description:"http client httpOptions: key value pairs, where key is one of the following: HTTP httpOptions:RequestTimeoutMs,TimeoutMs,KeepAliveTimeMs,TLSHandshakeTimeoutMs,ResponseHeaderTimeoutMs,MaxIdleConns,FollowRedirects"`
	httpOptions []*toolbox.HttpOptions
	Requests    []*Request
	Expect      map[string]interface{} `description:"If specified it will validated response as actual"`
}

//Init initializes send request
func (s *SendRequest) Init() error {

	if s.Expect == nil {
		s.Expect = make(map[string]interface{})
	}

	if len(s.Requests) == 0 {
		return nil
	}

	if len(s.Options) > 0 {
		s.httpOptions = make([]*toolbox.HttpOptions, 0)
		for k, v := range s.Options {
			s.httpOptions = append(s.httpOptions, &toolbox.HttpOptions{Key: k, Value: v})
		}
	}
	if _, has := s.Expect["Responses"]; has {
		for i, expect := range toolbox.AsSlice(s.Expect["Responses"]) {
			if i >= len(s.Requests) {
				break
			}
			s.Requests[i].Expect = toolbox.AsMap(expect)
		}
		return nil
	}

	var hasExpectedResponse = false
	var emptyMap = make(map[string]interface{})
	var expectedResponses = make([]interface{}, 0)
	for _, request := range s.Requests {
		if request.Expect != nil {
			hasExpectedResponse = true
			expectedResponses = append(expectedResponses, request.Expect)
		} else {
			expectedResponses = append(expectedResponses, emptyMap)
		}
	}
	if hasExpectedResponse {
		s.Expect["Responses"] = expectedResponses
	} else if len(s.Expect) == 0 {
		s.Expect = nil
	}
	return nil
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

//LoadRequest represents a send http request.
type LoadRequest struct {
	*SendRequest
	ThreadCount int    `description:"defines number of http client sending request concurrently, default 3"`
	Repeat      int    `description:"defines how many times repeat individual request, default 1"`
	AssertMod   int    `description:"defines modulo for assertion on repeated request (make sure you have enough memory)"`
	Message     string `description:"reporting message during stress test, the following is available: $load.[QPS|Count|Elapsed|Timeouts|Errors|Error]"`
}

func (r *LoadRequest) Init() error {
	if r.ThreadCount == 0 {
		r.ThreadCount = 3
	}
	if r.Repeat == 0 {
		r.Repeat = 1
	}
	if len(r.Requests) == 0 {
		return nil
	}

	for _, req := range r.Requests {
		if req.Repeater == nil {
			req.Repeater = req.Repeater.Init()
			req.Repeat = r.Repeat
		}
		if req.Repeat == 0 {
			req.Repeat = r.Repeat
		}
	}

	if r.AssertMod == 0 {
		r.AssertMod = 1024
	}

	if r.Message == "" {
		r.Message = " $load.Elapsed: Count: $load.Count, QPS: $load.QPS, Timeouts: $load.Timeouts, Errors: $load.Errors, Error: $load.Error"
	}
	return r.SendRequest.Init()
}

func (r *LoadRequest) Validate() error {
	if len(r.Requests) == 0 {
		return fmt.Errorf("requests were empty")
	}
	for _, request := range r.Requests {
		if request.When != "" {
			return fmt.Errorf("conditional execution is not supported in stress test mode")
		}
		if len(request.Variables) > 0 {
			return fmt.Errorf("scraping variables is not supported in stress test mode")
		}
		if len(request.Extract) > 0 {
			return fmt.Errorf("scraping data is not supported in stress test mode")
		}
	}

	return nil
}

//LoadRequest represents a stress test response
type LoadResponse struct {
	SendResponse
	Status              string
	Error               string
	QPS                 float64
	TimeoutCount        int
	ErrorCount          int
	StatusCodes         map[int]int
	TestDurationSec     float64
	RequestCount        int
	MinResponseTimeInMs float64
	AvgResponseTimeInMs float64
	MaxResponseTimeInMs float64
}
