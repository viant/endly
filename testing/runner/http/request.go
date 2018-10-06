package http

import (
	"github.com/viant/endly/model"
	"net/http"
	"github.com/viant/endly"
	"io"
	"github.com/viant/toolbox"
	"strings"
	"bytes"
	"github.com/viant/endly/udf"
	"github.com/viant/endly/util"
)

//ServiceRequest represents an http request
type Request struct {
	*model.Repeater
	When        string `description:"criteria to send this request"`
	Method      string `required:"true" description:"HTTP Method"`
	URL         string
	Header      http.Header
	Cookies     Cookies
	Body        string
	Replace     map[string]string `description:"response body key value pair replacement"`
	RequestUdf  string            `description:"user defined function in context.state key, i,e, json to protobuf"`
	ResponseUdf string            `description:"user defined function in context.state key, i,e, protobuf to json"`
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

//Build builds an http.Request
func (r *Request) Build(context *endly.Context, sessionCookies Cookies) (*http.Request, bool, error) {
	request:= r.Expand(context)
	var reader io.Reader
	var expectBinary = false
	var err error
	var ok bool
	if len(r.Body) > 0 {
		body := []byte(request.Body)
		if request.RequestUdf != "" {
			transformed, err := udf.TransformWithUDF(context, request.RequestUdf, request.URL, string(body))
			if err != nil {
				return  nil, false, err
			}
			if body, ok = transformed.([]byte); !ok {
				body = []byte(toolbox.AsString(transformed))
			}
		}
		expectBinary = strings.HasPrefix(string(body), "base64:")
		body, err = util.FromPayload(string(body))
		if err != nil {
			return  nil, expectBinary, err
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