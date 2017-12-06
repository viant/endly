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
	"time"
)

//HTTPRunnerServiceID represents http runner service id.
const HTTPRunnerServiceID = "http/runner"
const HttpRunnerExitCriteriaEventType = "HttpExitEvaluation"

type httpRunnerService struct {
	*AbstractService
}

func (s *httpRunnerService) processResponse(context *Context, sendRequest *SendHTTPRequest, sendHTTPRequest *HTTPRequest, response *HTTPResponse, httpResponse *http.Response, isBase64Encoded bool, extracted map[string]string) (string, error) {
	state := context.state
	response.Header = make(map[string][]string)
	copyHeaders(httpResponse.Header, response.Header)
	readBody(httpResponse, response, isBase64Encoded)
	if sendRequest.ResponseUdf != "" {
		var udf, has = UdfRegistry[sendRequest.ResponseUdf]
		if !has {
			return "", fmt.Errorf("failed to lookup udf: %v for: %v", sendRequest.ResponseUdf, sendHTTPRequest.URL)
		}
		transformed, err := udf(response.Body, state)
		if err != nil {
			return "", fmt.Errorf("failed to send sendRequest unable to run udf: %v", err)
		}
		response.Body = toolbox.AsString(transformed)
	}

	var responseBody = replaceResponseBodyIfNeeded(sendHTTPRequest, response.Body)
	sendHTTPRequest.Extraction.Extract(context, extracted, responseBody)
	return responseBody, nil
}

func (s *httpRunnerService) sendRequest(context *Context, client *http.Client, sendHTTPRequest *HTTPRequest, sessionCookies *Cookies, sendRequest *SendHTTPRequest, result *SendHTTPResponse) error {
	var err error
	var state = context.State()
	cookies := state.GetMap("cookies")
	var reader io.Reader
	var isBase64Encoded = false
	sendHTTPRequest = sendHTTPRequest.Expand(context)
	var body []byte
	if len(sendHTTPRequest.Body) > 0 {
		body = []byte(sendHTTPRequest.Body)
		if strings.HasPrefix(sendHTTPRequest.Body, "text:") {
			body = []byte(sendHTTPRequest.Body[5:])
		}
		if sendRequest.RequestUdf != "" {
			var udf, has = UdfRegistry[sendRequest.RequestUdf]
			if !has {
				return fmt.Errorf("failed to lookup udf: %v for: %v", sendRequest.RequestUdf, sendHTTPRequest.URL)
			}
			transformed, err := udf(sendHTTPRequest.Body, state)
			if err != nil {
				return fmt.Errorf("failed to send sendRequest unable to run udf: %v", err)
			}
			body = []byte(toolbox.AsString(transformed))
		}

		if strings.HasPrefix(string(body), "base64:") {
			isBase64Encoded = true
			reader = base64.NewDecoder(base64.StdEncoding, bytes.NewReader(body[7:]))
			body, err = ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("failed to decode base64 sendRequest: %v, %v", sendHTTPRequest.URL, err)
			}
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
	result.Responses = append(result.Responses, response)
	startEvent := s.Begin(context, sendHTTPRequest, Pairs("request", sendHTTPRequest))

	var repeat = sendHTTPRequest.Repeat
	if repeat == 0 {
		repeat = 1
	}
	var httpResponse *http.Response
	var responseBody string
	var bodyCache []byte
	var useCachedBody = repeat > 1 && httpRequest.ContentLength > 0
	if useCachedBody {
		bodyCache, err = ioutil.ReadAll(httpRequest.Body)
		if err != nil {
			return err
		}
	}

	for i := 0; i < repeat; i++ {
		if useCachedBody {
			httpRequest.Body = ioutil.NopCloser(bytes.NewReader(bodyCache))
		}
		httpResponse, err = client.Do(httpRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
			return nil
		}
		responseBody, err = s.processResponse(context, sendRequest, sendHTTPRequest, response, httpResponse, isBase64Encoded, result.Extracted)
		var extractedState = context.state.Clone()
		for k, v := range result.Extracted {
			extractedState[k] = v
		}
		critera := extractedState.ExpandAsText(sendHTTPRequest.ExitCriteria)
		canBreak, err := EvaluateCriteria(context, critera, HttpRunnerExitCriteriaEventType, false);
		if err != nil {
			return fmt.Errorf("failed to check http exit criteia: %v", err)
		}
		if canBreak {
			break;
		}
		if sendHTTPRequest.SleepTimeMs > 0 {
			timeToSleep  := time.Millisecond * time.Duration(sendHTTPRequest.SleepTimeMs)
			time.Sleep(timeToSleep)
		}
	}


	var responseCookies Cookies = httpResponse.Cookies()
	response.Cookies = responseCookies.IndexByName()
	for k, cookie := range response.Cookies {
		cookies.Put(k, cookie.Value)
	}
	sessionCookies.AddCookies(responseCookies...)

	endEvent := s.End(context)(startEvent, Pairs("response", response))

	var previous = state.GetMap("previous")
	if previous == nil {
		previous = data.NewMap()
	}
	response.Code = httpResponse.StatusCode
	response.TimeTakenMs = endEvent.TimeTakenMs
	if strings.HasPrefix(responseBody, "{") {
		response.JSONBody = make(map[string]interface{})
		err = toolbox.NewJSONDecoderFactory().Create(strings.NewReader(responseBody)).Decode(&response.JSONBody)
		if err == nil {
			sendHTTPRequest.Variables.Apply(data.Map(response.JSONBody), previous)
		}
	}

	for k, v := range result.Extracted {
		var expanded = previous.Expand(v)
		previous[k] = state.Expand(expanded)
	}
	sendHTTPRequest.Variables.Apply(previous, previous)

	if len(previous) > 0 {
		state.Put("previous", previous)
	}

	if sendHTTPRequest.MatchBody != "" {
		return nil
	}
	for _, candidate := range sendRequest.Requests {
		if candidate.MatchBody != "" && strings.Contains(response.Body, candidate.MatchBody) {
			err = s.sendRequest(context, client, candidate, sessionCookies, sendRequest, result)
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
		encoder.Write(body)
		encoder.Close()
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

func (s *httpRunnerService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *SendHTTPRequest:
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
	case "send":
		return &SendHTTPRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewHTTPpRunnerService creates a new http runner service
func NewHTTPpRunnerService() Service {
	var result = &httpRunnerService{
		AbstractService: NewAbstractService(HTTPRunnerServiceID),
	}
	result.AbstractService.Service = result
	return result
}
