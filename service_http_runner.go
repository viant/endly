package endly

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

//HTTPRunnerServiceID represents http runner service id.
const HTTPRunnerServiceID = "http/runner"

//HTTPRunnerServiceSendAction represents a send action.
const HTTPRunnerServiceSendAction = "send"

//HTTPRunner represent HttpExitEvaluation event name
const HTTPRunner = "HttpRunner"

//HTTPPreviousTripStateKey keys to store previous request details for multi trip HTTP Send request in context state
const HTTPPreviousTripStateKey = "previous"

type httpRunnerService struct {
	*AbstractService
}

func (s *httpRunnerService) processResponse(context *Context, sendRequest *SendHTTPRequest, sendHTTPRequest *HTTPRequest, response *HTTPResponse, httpResponse *http.Response, isBase64Encoded bool, extracted map[string]string) (string, error) {
	response.Header = make(map[string][]string)
	copyHeaders(httpResponse.Header, response.Header)
	readBody(httpResponse, response, isBase64Encoded)
	if sendHTTPRequest.ResponseUdf != "" {
		transformed, err := transformWithUDF(context, sendHTTPRequest.ResponseUdf, sendHTTPRequest.URL, response.Body)
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

func (s *httpRunnerService) sendRequest(context *Context, client *http.Client, sendHTTPRequest *HTTPRequest, sessionCookies *Cookies, sendGroupRequest *SendHTTPRequest, sendGroupResponse *SendHTTPResponse) error {
	var err error
	var state = context.State()
	cookies := state.GetMap("cookies")
	var reader io.Reader
	var isBase64Encoded = false
	sendHTTPRequest = sendHTTPRequest.Expand(context)
	var body []byte
	var ok bool
	if len(sendHTTPRequest.Body) > 0 {
		body = []byte(sendHTTPRequest.Body)
		if sendHTTPRequest.RequestUdf != "" {
			transformed, err := transformWithUDF(context, sendHTTPRequest.RequestUdf, sendHTTPRequest.URL, string(body))
			if err != nil {
				return err
			}
			if body, ok = transformed.([]byte); !ok {
				body = []byte(toolbox.AsString(transformed))
			}
		}
		isBase64Encoded = strings.HasPrefix(string(body), "base64:")
		body, err = FromPayload(string(body))
		if err != nil {
			return err
		}
		reader = bytes.NewReader(body)
	}

	httpRequest, err := http.NewRequest(strings.ToUpper(sendHTTPRequest.Method), sendHTTPRequest.URL, reader)
	if err != nil {
		return err
	}

	copyHeaders(sendHTTPRequest.Header, httpRequest.Header)
	sessionCookies.SetHeader(sendHTTPRequest.Header)
	sendHTTPRequest.Cookies.SetHeader(httpRequest.Header)

	response := &HTTPResponse{}
	sendGroupResponse.Responses = append(sendGroupResponse.Responses, response)
	startEvent := s.Begin(context, sendHTTPRequest, Pairs("request", sendHTTPRequest))

	repeatable := sendHTTPRequest.Repeatable.Get()

	var httpResponse *http.Response
	var responseBody string
	var bodyCache []byte
	var useCachedBody = repeatable.Repeat > 1 && httpRequest.ContentLength > 0
	if useCachedBody {
		bodyCache, err = ioutil.ReadAll(httpRequest.Body)
		if err != nil {
			return err
		}
	}

	handler := func() (interface{}, error) {
		if useCachedBody {
			httpRequest.Body = ioutil.NopCloser(bytes.NewReader(bodyCache))
		}
		httpResponse, err = client.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		responseBody, err = s.processResponse(context, sendGroupRequest, sendHTTPRequest, response, httpResponse, isBase64Encoded, sendGroupResponse.Extracted)
		return responseBody, err
	}

	err = repeatable.Run(s.AbstractService, HTTPRunner, context, handler, sendGroupResponse.Extracted)
	if err != nil {
		return err
	}
	var responseCookies Cookies = httpResponse.Cookies()
	response.Cookies = responseCookies.IndexByName()
	for k, cookie := range response.Cookies {
		cookies.Put(k, cookie.Value)
	}
	sessionCookies.AddCookies(responseCookies...)

	endEvent := s.End(context)(startEvent, Pairs("response", response))

	var previous = state.GetMap(HTTPPreviousTripStateKey)
	if previous == nil {
		previous = data.NewMap()
	}
	response.Code = httpResponse.StatusCode
	response.TimeTakenMs = endEvent.TimeTakenMs

	if toolbox.IsCompleteJSON(responseBody) {
		response.JSONBody, err = toolbox.JSONToMap(responseBody)
		if err == nil && sendHTTPRequest.Repeatable != nil {
			_ = sendHTTPRequest.Variables.Apply(data.Map(response.JSONBody), previous)
		}
	}

	for k, v := range sendGroupResponse.Extracted {
		var expanded = previous.Expand(v)
		previous[k] = state.Expand(expanded)
	}

	err = repeatable.Variables.Apply(previous, previous)
	if err != nil {
		return err
	}
	if len(previous) > 0 {
		state.Put(HTTPPreviousTripStateKey, previous)
	}
	if sendHTTPRequest.MatchBody != "" {
		return nil
	}

	for _, candidate := range sendGroupRequest.Requests {
		if candidate.MatchBody != "" && strings.Contains(response.Body, candidate.MatchBody) {
			err = s.sendRequest(context, client, candidate, sessionCookies, sendGroupRequest, sendGroupResponse)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func replaceResponseBodyIfNeeded(sendHTTPRequest *HTTPRequest, responseBody string) string {
	if len(sendHTTPRequest.Replace) > 0 {
		for k, v := range sendHTTPRequest.Replace {
			responseBody = strings.Replace(responseBody, k, v, len(responseBody))
		}
	}
	return responseBody
}

func (s *httpRunnerService) send(context *Context, request *SendHTTPRequest) (*SendHTTPResponse, error) {
	client, err := toolbox.NewHttpClient(request.Options...)
	if err != nil {
		return nil, fmt.Errorf("failed to send req: %v", err)
	}
	var result = &SendHTTPResponse{
		Responses: make([]*HTTPResponse, 0),
		Extracted: make(map[string]string),
	}
	var sessionCookies Cookies = make([]*http.Cookie, 0)
	var state = context.State()
	if !state.Has("cookies") {
		state.Put("cookies", data.NewMap())
	}
	for _, req := range request.Requests {
		if req.MatchBody != "" {
			continue
		}
		err = s.sendRequest(context, client, req, &sessionCookies, request, result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil

}
func readBody(httpResponse *http.Response, response *HTTPResponse, expectBased64Encoded bool) {

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

func copyHeaders(source http.Header, target http.Header) {
	for key, values := range source {
		if _, has := target[key]; !has {
			target[key] = make([]string, 0)
		}
		if len(values) == 1 {
			target.Set(key, values[0])
		} else {

			for _, value := range values {
				target.Add(key, value)
			}
		}
	}
}

func copyExpandedHeaders(source http.Header, target http.Header, context *Context) {
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

//resetContext resets context for variables with Reset flag set, and removes HTTPPreviousTripStateKey
func (s *httpRunnerService) resetContext(context *Context, request *SendHTTPRequest) {
	state := context.state
	state.Delete(HTTPPreviousTripStateKey)
	for _, request := range request.Requests {
		if request.Repeatable != nil && len(request.Extraction) > 0 {
			request.Extraction.Reset(state)
		}
	}
}

func (s *httpRunnerService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *SendHTTPRequest:
		defer s.resetContext(context, actualRequest)
		response.Response, err = s.send(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to send request: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *httpRunnerService) NewRequest(action string) (interface{}, error) {
	switch action {
	case HTTPRunnerServiceSendAction:
		return &SendHTTPRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func (s *httpRunnerService) NewResponse(action string) (interface{}, error) {
	switch action {
	case HTTPRunnerServiceSendAction:
		return &SendHTTPResponse{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

//NewHTTPpRunnerService creates a new http runner service
func NewHTTPpRunnerService() Service {
	var result = &httpRunnerService{
		AbstractService: NewAbstractService(HTTPRunnerServiceID,
			HTTPRunnerServiceSendAction),
	}
	result.AbstractService.Service = result
	return result
}
