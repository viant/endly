package http

import (
	"net/http"

	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

//SendRequest represents a send http request.
type SendRequest struct {
	Options  []*toolbox.HttpOptions `description:"http client options: key value pairs, where key is one of the following: HTTP options:RequestTimeoutMs,TimeoutMs,KeepAliveTimeMs,TLSHandshakeTimeoutMs,ResponseHeaderTimeoutMs,MaxIdleConns"`
	Requests []*Request
}

//Request represents an http request
type Request struct {
	*endly.Repeater
	When        string `description:"condition expression that will be evaluated successfully before firing this request"`
	Method      string `required:"true" description:"HTTP Method"`
	URL         string
	Header      http.Header
	Cookies     Cookies
	Body        string
	Replace     map[string]string `description:"response body key value pair replacement"`
	RequestUdf  string            `description:"user defined function in context.state key, i,e, json to protobuf"`
	ResponseUdf string            `description:"user defined function in context.state key, i,e, protobuf to json"`
}

//SendResponse represnets a send response
type SendResponse struct {
	Responses []*Response
	Data      data.Map
}

//Response represents Http response
type Response struct {
	//Request     *Request
	Code        int
	Header      http.Header
	Cookies     map[string]*http.Cookie
	Body        string
	JSONBody    map[string]interface{} `description:"structure data if Body was JSON"`
	TimeTakenMs int
	Error       string
}

//Cookies represents cookie
type Cookies []*http.Cookie

//SetHeader sets cookie header
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

//IndexByName index cookie by name
func (c *Cookies) IndexByName() map[string]*http.Cookie {
	var result = make(map[string]*http.Cookie)
	for _, cookie := range *c {
		result[cookie.Name] = cookie
	}
	return result
}

//IndexByPosition index cookie by position
func (c *Cookies) IndexByPosition() map[string]int {
	var result = make(map[string]int)
	for i, cookie := range *c {
		result[cookie.Name] = i
	}
	return result
}

//AddCookies adds cookies
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

//Expand substitute request data with matching context map state.
func (r *Request) Expand(context *endly.Context) *Request {
	header := make(map[string][]string)
	copyExpandedHeaders(r.Header, header, context)
	return &Request{
		When:        context.Expand(r.When),
		Method:      r.Method,
		URL:         context.Expand(r.URL),
		Body:        context.Expand(r.Body),
		Header:      header,
		Repeater:    r.Repeater,
		Replace:     r.Replace,
		RequestUdf:  r.RequestUdf,
		ResponseUdf: r.ResponseUdf,
	}
}

// Method to evaluate if a request condition is satisfied. This may be used as a pre-requisite before making the actual request call
// true on success or when no condition is present. false on failed evaluation. Error
func (r *Request) EvaluateWhen(context *endly.Context) (isEvaluated bool, err error) {
	isEvaluated = true //By default
	if r.When != "" {
		isEvaluated, err = endly.Evaluate(context, context.State(), r.When, "Evaluate When condition with expersion"+r.When, isEvaluated)
	}
	return
}
