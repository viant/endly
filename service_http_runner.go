package endly

import (
	"net/http"
	"time"
	"fmt"
	"github.com/viant/toolbox"
	"bytes"
	"strings"
	"io"
	"io/ioutil"
	"unicode"
	"unicode/utf8"
	"encoding/base64"
	"github.com/viant/endly/common"
)

const HttpRunnerServiceId = " http/runner"

type Cookies []*http.Cookie

func (c *Cookies) SetHeader(header http.Header) {
	if len(*c) == 0 {
		return
	}
	for _, cookie := range (*c) {
		if v := cookie.String(); v != "" {
			header.Add("Cookie", v)
		}
	}
}

func (c *Cookies) IndexByName() (map[string]*http.Cookie) {

	var result = make(map[string]*http.Cookie)

	for _, cookie := range *c {
		result[cookie.Name] = cookie
	}
	return result
}

func (c *Cookies) IndexByPosition() (map[string]int) {
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
	Method     string
	URL        string
	Header     http.Header
	Cookies    Cookies
	Body       string
	Extraction DataExtractions
}

type HttpResponse struct {
	Request     *HttpRequest
	Code        int
	Header      http.Header
	Cookies     map[string]*http.Cookie
	Body        string
	TimeTakenMs int
	Error       string
}

type SendRequest struct {
	Options  []*toolbox.HttpOptions
	Requests []*HttpRequest
}

type SendResponse struct {
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

func (s *httpRunnerService) send(context *Context, request *SendRequest) (*SendResponse, error) {
	client, err := toolbox.NewHttpClient(request.Options...)
	if err != nil {
		return nil, fmt.Errorf("Failed to send req: %v", err)
	}

	var result = &SendResponse{
		Responses: make([]*HttpResponse, 0),
		Extracted: make(map[string]string),
	}

	var sessionCookies Cookies = make([]*http.Cookie, 0)
	var state = context.State()

	if ! state.Has("cookies") {
		state.Put("cookies", common.NewMap())
	}
	cookies := state.GetMap("cookies")
	for _, req := range request.Requests {

		var reader io.Reader;
		if len(req.Body) > 0 {
			reader = strings.NewReader(req.Body)
		}
		httpRequest, err := http.NewRequest(strings.ToUpper(req.Method), req.URL, reader)
		if err != nil {
			return nil, err
		}
		httpRequest.Header = make(map[string][]string)
		copyHeaders(req.Header, httpRequest.Header)
		sessionCookies.SetHeader(httpRequest.Header)
		req.Cookies.SetHeader(httpRequest.Header)

		response := &HttpResponse{
			Request: req,
		}
		result.Responses = append(result.Responses, response)

		startTime := time.Now()
		httpResponse, err := client.Do(httpRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
			break
		}
		endTime := time.Now()
		response.Header = make(map[string][]string)
		copyHeaders(httpResponse.Header, response.Header)
		readBody(httpResponse, response)
		req.Extraction.Extract(context, result.Extracted, response.Body)
		var responseCookies Cookies = httpResponse.Cookies()


		response.Cookies = responseCookies.IndexByName()
		for k, cookie := range response.Cookies {
			cookies.Put(k, cookie.Value)
		}
		sessionCookies.AddCookies(responseCookies...)
		response.Code = httpResponse.StatusCode
		response.TimeTakenMs = int((endTime.UnixNano() - startTime.UnixNano()) / int64(time.Millisecond))
	}
	return result, nil

}
func readBody(httpResponse *http.Response, response *HttpResponse) {

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		return
	}
	httpResponse.Body.Close()
	if (isBinary(body)) {
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
		if _, has:=target[key];!has {
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

func (s *httpRunnerService) Run(context *Context, request interface{}) *Response {
	var response = &Response{
		Status: "ok",
	}
	switch castedRequest := request.(type) {
	case *SendRequest:
		response.Response, response.Error = s.send(context, castedRequest)
	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}

	if response.Error != nil {
		response.Status = "error"
	}
	return response
}

func (s *httpRunnerService) NewRequest(name string) (interface{}, error) {
	return &SendRequest{}, nil
}

func NewHttpRunnerService() Service {
	var result = &httpRunnerService{
		AbstractService: NewAbstractService(HttpRunnerServiceId),
	}
	result.AbstractService.Service = result
	return result
}
