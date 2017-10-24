package endly

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

const HttpRunnerServiceId = "http/runner"

type Cookies []*http.Cookie

func (c *Cookies) SetHeader(header http.Header) {
	if len(*c) == 0 {
		return
	}
	for _, cookie := range *c {
		if v := cookie.String(); v != "" {
			header.Add("Cookie", v)
		}
	}
}

func (c *Cookies) IndexByName() map[string]*http.Cookie {

	var result = make(map[string]*http.Cookie)

	for _, cookie := range *c {
		result[cookie.Name] = cookie
	}
	return result
}

func (c *Cookies) IndexByPosition() map[string]int {
	var result = make(map[string]int)
	for i, cookie := range *c {
		result[cookie.Name] = i
	}
	return result
}

func (c *Cookies) AddCookies(cookies ...*http.Cookie) {
	if len(cookies) == 0 {
		return
	}
	var indexed = c.IndexByPosition()
	for _, cookie := range cookies {

		if position, has := indexed[cookie.Name]; has {
			(*c)[position] = cookie
		} else {
			*c = append(*c, cookie)
		}
	}
}

type HttpRequest struct {
	MatchBody  string //only run this execution is output from a previous command is matched
	Method     string
	URL        string
	Header     http.Header
	Cookies    Cookies
	Body       string
	Extraction DataExtractions
}

func (r *HttpRequest) Expand(context *Context) *HttpRequest {
	header := make(map[string][]string)

	copyExpandedHeaders(r.Header, header, context)
	return &HttpRequest{
		MatchBody:  context.Expand(r.MatchBody),
		Method:     r.Method,
		URL:        context.Expand(r.URL),
		Body:       context.Expand(r.Body),
		Header:     header,
		Extraction: r.Extraction,
	}
}

type HttpResponse struct {
	//Request     *HttpRequest
	Code        int
	Header      http.Header
	Cookies     map[string]*http.Cookie
	Body        string
	TimeTakenMs int
	Error       string
}

type SendHttpRequest struct {
	Options     []*toolbox.HttpOptions
	Requests    []*HttpRequest
	RequestUdf  string
	ResponseUdf string
}

type SendHttpResponse struct {
	Responses []*HttpResponse
	Extracted map[string]string
}

type httpRunnerService struct {
	*AbstractService
}

func isBinary(input []byte) bool {
	for i, w := 0, 0; i < len(input); i += w {
		runeValue, width := utf8.DecodeRune(input[i:])
		if unicode.IsControl(runeValue) {
			return true
		}
		w = width
	}
	return false
}

func (s *httpRunnerService) sendRequest(context *Context, client *http.Client, sendHttpRequest *HttpRequest, sessionCookies *Cookies, sendRequest *SendHttpRequest, result *SendHttpResponse) error {
	var err error
	var state = context.State()
	cookies := state.GetMap("cookies")
	var reader io.Reader
	var isBase64Encoded = false
	sendHttpRequest = sendHttpRequest.Expand(context)
	var body []byte
	if len(sendHttpRequest.Body) > 0 {
		body = []byte(sendHttpRequest.Body)
		if strings.HasPrefix(sendHttpRequest.Body, "text:") {
			body = []byte(sendHttpRequest.Body[5:])
		}

		if sendRequest.RequestUdf != "" {
			var udf, has = UdfRegistry[sendRequest.RequestUdf]
			if !has {
				return fmt.Errorf("Failed to lookup udf: %v for: %v\n", sendRequest.RequestUdf, sendHttpRequest.URL)
			}
			transformed, err := udf(sendHttpRequest.Body, state)
			if err != nil {
				return fmt.Errorf("Failed to send sendRequest unable to run udf: %v\n", err)
			}
			body = []byte(toolbox.AsString(transformed))
		}

		if strings.HasPrefix(string(body), "base64:") {
			isBase64Encoded = true
			reader = base64.NewDecoder(base64.StdEncoding, bytes.NewReader(body[7:]))
			body, err = ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("Failed to decode base64 sendRequest: %v, %v", sendHttpRequest.URL, err)
			}
		}
		reader = bytes.NewReader(body)
	}

	httpRequest, err := http.NewRequest(strings.ToUpper(sendHttpRequest.Method), sendHttpRequest.URL, reader)
	if err != nil {
		return err
	}

	copyHeaders(httpRequest.Header, httpRequest.Header)
	sessionCookies.SetHeader(httpRequest.Header)
	sendHttpRequest.Cookies.SetHeader(httpRequest.Header)
	response := &HttpResponse{
		//Request: sendHttpRequest,
	}
	result.Responses = append(result.Responses, response)
	startEvent := s.Begin(context, sendHttpRequest, Pairs("request", sendHttpRequest))
	httpResponse, err := client.Do(httpRequest)

	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		return nil
	}

	response.Header = make(map[string][]string)
	copyHeaders(httpResponse.Header, response.Header)

	readBody(httpResponse, response, isBase64Encoded)

	if sendRequest.ResponseUdf != "" {
		var udf, has = UdfRegistry[sendRequest.ResponseUdf]
		if !has {
			return fmt.Errorf("Failed to lookup udf: %v for: %v\n", sendRequest.ResponseUdf, sendHttpRequest.URL)
		}
		transformed, err := udf(response.Body, state)
		if err != nil {
			return fmt.Errorf("Failed to send sendRequest unable to run udf: %v\n", err)
		}
		response.Body = toolbox.AsString(transformed)
	}
	endEvent := s.End(context)(startEvent, Pairs("response", response))

	sendHttpRequest.Extraction.Extract(context, result.Extracted, response.Body)
	var responseCookies Cookies = httpResponse.Cookies()

	response.Cookies = responseCookies.IndexByName()
	for k, cookie := range response.Cookies {
		cookies.Put(k, cookie.Value)
	}
	sessionCookies.AddCookies(responseCookies...)

	response.Code = httpResponse.StatusCode
	response.TimeTakenMs = endEvent.TimeTakenMs
	for _, candidate := range sendRequest.Requests {
		if candidate.MatchBody != "" && strings.Contains(response.Body, candidate.MatchBody) {
			return s.sendRequest(context, client, candidate, sessionCookies, sendRequest, result)
		}
	}
	return nil
}

func (s *httpRunnerService) send(context *Context, request *SendHttpRequest) (*SendHttpResponse, error) {
	client, err := toolbox.NewHttpClient(request.Options...)
	if err != nil {
		return nil, fmt.Errorf("Failed to send req: %v", err)
	}
	var result = &SendHttpResponse{
		Responses: make([]*HttpResponse, 0),
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
func readBody(httpResponse *http.Response, response *HttpResponse, expectBased64Encoded bool) {

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
	case *SendHttpRequest:
		response.Response, err = s.send(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to send request: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *httpRunnerService) NewRequest(action string) (interface{}, error) {
	return &SendHttpRequest{}, nil
}

func NewHttpRunnerService() Service {
	var result = &httpRunnerService{
		AbstractService: NewAbstractService(HttpRunnerServiceId),
	}
	result.AbstractService.Service = result
	return result
}
