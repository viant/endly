package http

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox"
	"net/http"
	"time"
)

//ServiceID represents http runner service id.
const ServiceID = "http/runner"
const RunnerID = "HttpRunner"

type service struct {
	*endly.AbstractService
}

func (s *service) send(context *endly.Context, sendGroupRequest *SendRequest) (*SendResponse, error) {
	client, err := toolbox.NewHttpClient(s.applyDefaultTimeoutIfNeeded(sendGroupRequest.Options)...)
	if err != nil {
		return nil, fmt.Errorf("failed to send req: %v", err)
	}
	initializeContext(context)
	defer s.resetContext(context, sendGroupRequest)

	if err = udf.RegisterProviders(sendGroupRequest.UdfProviders); err != nil {
		return nil, err
	}

	var sendGroupResponse = &SendResponse{
		Responses: make([]*Response, 0),
		Data:      make(map[string]interface{}),
	}
	var sessionCookies Cookies = make([]*http.Cookie, 0)
	for _, req := range sendGroupRequest.Requests {
		err = s.sendRequest(context, client, req, &sessionCookies, sendGroupRequest, sendGroupResponse)
		if err != nil {
			return nil, err
		}
	}
	if sendGroupRequest.Expect != nil {
		var actual = map[string]interface{}{
			"Responses": sendGroupResponse.Responses,
			"Data":      sendGroupResponse.Data,
		}
		sendGroupResponse.Assert, err = validator.Assert(context, sendGroupRequest, sendGroupRequest.Expect, actual, "HTTP.responses", "assert http responses")
	}
	return sendGroupResponse, nil

}

func (s *service) sendRequest(context *endly.Context, client *http.Client, request *Request, sessionCookies *Cookies, sendGroupRequest *SendRequest, sendGroupResponse *SendResponse) error {
	var err error
	var state = context.State()
	cookies := state.GetMap("cookies")
	trips := Trips(state.GetMap(TripsKey))
	canRun, err := criteria.Evaluate(context, context.State(), request.When, fmt.Sprintf("%v.When", "HttpRequest"), true)
	if err != nil || !canRun {
		return err
	}
	httpRequest, expectBinary, err := request.Build(context, *sessionCookies)
	if err != nil {
		return err
	}
	trips.addRequest(request)
	startEvent := s.Begin(context, request)
	repeater := request.Repeater.Init()
	var response *Response
	bodyProvider, err := getRequestBodyReader(httpRequest, repeater.Repeat)

	handler := func() (interface{}, error) {
		httpRequest.Body = bodyProvider()
		httpResponse, err := client.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		if response == nil { //if request is repeated only the allocated one, and keep overriding it to see the last snapshot
			response = sendGroupResponse.NewResponse()
		}
		response.Merge(httpResponse, expectBinary)
		response.UpdateCookies(cookies)
		sessionCookies.AddCookies(httpResponse.Cookies()...)
		err = response.TransformBodyIfNeeded(context, request)
		return response.Body, err
	}

	err = repeater.Run(s.AbstractService, RunnerID, context, handler, sendGroupResponse.Data)
	if err != nil {
		return err
	}
	if toolbox.IsCompleteJSON(response.Body) {
		response.JSONBody, err = toolbox.JSONToMap(response.Body)
	}
	trips.setData(sendGroupResponse.Data)
	trips.addResponse(response)
	endEvent := s.End(context)(startEvent, response)
	response.TimeTakenMs = int(endEvent.Timestamp().Sub(startEvent.Timestamp()) / time.Millisecond)
	return nil
}

func (s *service) applyDefaultTimeoutIfNeeded(options []*toolbox.HttpOptions) []*toolbox.HttpOptions {
	if len(options) > 0 {
		return options
	}
	return []*toolbox.HttpOptions{
		{
			Key:   "RequestTimeoutMs",
			Value: 120000,
		},
		{
			Key:   "TimeoutMs",
			Value: 120000,
		},
	}
}

//resetContext resets context for variables with Reset flag set, and removes PreviousTripStateKey
func (s *service) resetContext(context *endly.Context, request *SendRequest) {
	state := context.State()
	state.Delete(TripsKey)
	for _, request := range request.Requests {
		if request.Repeater != nil && len(request.Extraction) > 0 {
			request.Extraction.Reset(state)
		}
	}
}

const httpRunnerSendRequestExample = `{
  "Requests": [
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k1=v1\u0026k2=v2"
    },
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k10=v1\u0026k2=v2"
    },
    {
      "When": "",
      "Method": "POST",
      "URL": "http://127.0.0.1:8777/event4/",
      "Body": "Lorem Ipsum is simply dummy text of the printing and typesetting industry."
    }
  ]
}`

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "send",
		RequestInfo: &endly.ActionInfo{
			Description: "send http request(s)",
			Examples: []*endly.UseCase{
				{
					Description: "send",
					Data:        httpRunnerSendRequestExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SendRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SendResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SendRequest); ok {

				return s.send(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new http runner service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
