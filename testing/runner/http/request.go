package http

import (
	"bytes"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/udf"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io"
	"net/http"
	"strings"
)

//ServiceRequest represents an http request
type Request struct {
	*model.Repeater
	When           string `description:"criteria to send this request"`
	Method         string `required:"true" description:"HTTP Method"`
	URL         string
	Header      http.Header
	Cookies     Cookies
	Body        string
	JSONBody    interface{}            `description:"body JSON representation"`
	Replace     map[string]string      `description:"response body key value pair replacement"`
	RequestUdf  string                 `description:"user defined function in context.state key, i,e, json to protobuf"`
	ResponseUdf string                 `description:"user defined function in context.state key, i,e, protobuf to json"`
	DataSource  string                 `description:"variable input: response or response.body by default"`
	Expect      map[string]interface{} `description:"desired http response"`
}

//Clone substitute request data with matching context map state.
func (r *Request) Clone(context *endly.Context) *Request {
	header := make(map[string][]string)
	copyExpandedHeaders(r.Header, header, context)
	return &Request{
		When:        context.Expand(r.When),
		Method:      r.Method,
		URL:         context.Expand(r.URL),
		Body:        context.Expand(r.Body),
		JSONBody:    r.JSONBody,
		Header:      header,
		Repeater:    r.Repeater,
		Replace:     r.Replace,
		RequestUdf:  r.RequestUdf,
		ResponseUdf: r.ResponseUdf,
		DataSource:  r.DataSource,
	}
}

//Clone substitute request data with matching context map state.
func (r *Request) Expand(state data.Map) {
	r.URL = state.ExpandAsText(r.URL)
	r.Body = state.ExpandAsText(r.Body)
}

//Build builds an http.Request
func (r *Request) Build(context *endly.Context, sessionCookies Cookies) (*http.Request, bool, error) {
	if r.Body == "" && r.JSONBody != nil {
		var err error
		r.Body, err = toolbox.AsJSONText(r.JSONBody)
		if err != nil {
			return nil, false, err
		}
	}
	request := r.Clone(context)
	var reader io.Reader
	var expectBinary = false
	var err error
	var ok bool
	if len(r.Body) > 0 {
		body := []byte(request.Body)
		if request.RequestUdf != "" {
			transformed, err := udf.TransformWithUDF(context, request.RequestUdf, request.URL, string(body))
			if err != nil {
				return nil, false, err
			}
			if body, ok = transformed.([]byte); !ok {
				body = []byte(toolbox.AsString(transformed))
			}
		}
		expectBinary = strings.HasPrefix(string(body), "base64:")
		body, err = util.FromPayload(string(body))
		if err != nil {
			return nil, expectBinary, err
		}
		reader = bytes.NewReader(body)
	}

	httpRequest, err := http.NewRequest(strings.ToUpper(request.Method), request.URL, reader)
	if err != nil {
		return nil, expectBinary, err
	}

	copyHeaders(request.Header, httpRequest.Header)
	//Set cookies from active session
	SetCookies(sessionCookies, request.Header)
	//Set cookies from user http request
	SetCookies(request.Cookies, httpRequest.Header)
	return httpRequest, expectBinary, nil
}
