package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

//ServiceID represents http runner service id.
const ServiceID = "http/runner"

//PreviousTripStateKey keys to store previous request details for multi trip HTTP Send request in context state
const PreviousTripStateKey = "previous"

type service struct {
	*endly.AbstractService
}

func (s *service) send(context *endly.Context, request *SendRequest) (*SendResponse, error) {
	client, err := toolbox.NewHttpClient(request.Options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client for http runner service: %v", err)
	}
	defer s.resetContext(context, request)
	initializeContext(context)

	var sendGroupResponse = &SendResponse{
		Responses: make([]*Response, 0),
		Data:      make(map[string]interface{}),
	}

	for _, req := range request.Requests {
		err = s.sendRequest(context, client, req, request, sendGroupResponse)
		if err != nil {
			return nil, err
		}
	}
	return sendGroupResponse, nil

}

func (s *service) sendRequest(context *endly.Context, client *http.Client, sendRequest *Request, sendGroupRequest *SendRequest, sendGroupResponse *SendResponse) error {
	// Check if request can be executed
	if isValid, err := sendRequest.EvaluateWhen(context); !isValid {
		if err != nil {
			return err
		}
		return nil
	}

	//Build http request from SendRequest
	httpRequest, err := newRequestBuilder().setContext(context).setRequest(sendRequest).build()
	if err != nil {
		return err
	}

	//Add request to context trips
	t := trips(context.State().GetMap(Trips))
	t.addRequest(sendRequest)

	//Event tracking
	startEvent := s.Begin(context, sendRequest)

	response := &Response{}
	sendGroupResponse.Responses = append(sendGroupResponse.Responses, response)

	repeater := sendRequest.Repeater

	var HTTPResponse *http.Response
	var responseBody string

	handler := func() (interface{}, error) {
		HTTPResponse, err = client.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		var isBase64Encoded bool
		if httpRequest.TransferEncoding != nil && len(httpRequest.TransferEncoding) > 0 && httpRequest.TransferEncoding[0] == "base64" {
			isBase64Encoded = true
		}
		responseBody, err = s.processResponse(context, sendGroupRequest, sendRequest, response, HTTPResponse, isBase64Encoded, sendGroupResponse.Data)
		return responseBody, err
	}

	err = repeater.Run(s.AbstractService, "HTTPRunner", context, handler, sendGroupResponse.Data)
	if err != nil {
		return err
	}
	var responseCookies Cookies = HTTPResponse.Cookies()
	response.Cookies = responseCookies.IndexByName()
	for k, cookie := range response.Cookies {
		cookies.Put(k, cookie.Value)
	}
	sessionCookies.AddCookies(responseCookies...)
	endEvent := s.End(context)(startEvent, response)

	var previous = state.GetMap(PreviousTripStateKey)
	if previous == nil {
		previous = data.NewMap()
	}

	response.Code = HTTPResponse.StatusCode
	response.TimeTakenMs = int(startEvent.Timestamp().Sub(endEvent.Timestamp()) / time.Millisecond)

	if toolbox.IsCompleteJSON(responseBody) {
		response.JSONBody, err = toolbox.JSONToMap(responseBody)
		if err == nil && sendRequest.Repeater != nil {
			_ = sendRequest.Variables.Apply(data.Map(response.JSONBody), previous)
		}
	}

	for k, v := range sendGroupResponse.Data {
		var expanded = previous.Expand(v)
		previous[k] = state.Expand(expanded)
	}

	err = repeater.Variables.Apply(previous, previous)
	if err != nil {
		return err
	}
	if len(previous) > 0 {
		state.Put(PreviousTripStateKey, previous)
	}

	if sendGroupResponse.Responses != nil {
		var resp = make([]map[string]interface{}, 0)
		err = toolbox.DefaultConverter.AssignConverted(&resp, sendGroupResponse.Responses)
		if err != nil {
			return err
		}
		state.Put(PreviousTripStateKey, resp)
	}
	return nil
}

func (s *service) processResponse(context *endly.Context, sendRequest *SendRequest, sendHTTPRequest *Request, response *Response, httpResponse *http.Response, isBase64Encoded bool, extracted map[string]interface{}) (string, error) {
	response.Header = make(map[string][]string)
	copyHeaders(httpResponse.Header, response.Header)

	readBody(httpResponse, response, isBase64Encoded)
	if sendHTTPRequest.ResponseUdf != "" {
		transformed, err := udf.TransformWithUDF(context, sendHTTPRequest.ResponseUdf, sendHTTPRequest.URL, response.Body)
		if err != nil {
			return "", err
		}
		if toolbox.IsMap(transformed) {
			response.Body, _ = toolbox.AsJSONText(transformed)
		} else {
			response.Body = toolbox.AsString(transformed)
		}
	}

	var responseBody = replaceResponseBodyIfNeeded(sendHTTPRequest, response.Body)
	return responseBody, nil
}

func replaceResponseBodyIfNeeded(sendHTTPRequest *Request, responseBody string) string {
	if len(sendHTTPRequest.Replace) > 0 {
		for k, v := range sendHTTPRequest.Replace {
			responseBody = strings.Replace(responseBody, k, v, len(responseBody))
		}
	}
	return responseBody
}

func readBody(httpResponse *http.Response, response *Response, expectBased64Encoded bool) {

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		return
	}
	httpResponse.Body.Close()
	if expectBased64Encoded {
		buf := new(bytes.Buffer)
		encoder := base64.NewEncoder(base64.StdEncoding, buf)
		_, _ = encoder.Write(body)
		_ = encoder.Close()
		response.Body = "base64:" + string(buf.Bytes())

	} else {
		response.Body = string(body)
	}
}

func copyExpandedHeaders(source http.Header, target http.Header, context *endly.Context) {
	for key, values := range source {
		if _, has := target[key]; !has {
			target[key] = make([]string, 0)
		}
		if len(values) == 1 {
			target.Set(key, context.Expand(values[0]))
		} else {
			for _, value := range values {
				target.Add(key, context.Expand(value))
			}
		}
	}
}

func initializeContext(c *endly.Context) {
	var state = c.State()
	if !state.Has("cookies") {
		state.Put("cookies", data.NewMap())
	}
	if !state.Has(Trips) {
		state.Put(Trips, newTrips())
	}
}

//resetContext resets context for variables with Reset flag set, and removes PreviousTripStateKey
func (s *service) resetContext(context *endly.Context, request *SendRequest) {
	state := context.State()
	state.Delete(PreviousTripStateKey)
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
