package http

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

// Build a native http request from SendRequest
type RequestBuilder interface {
	setContext(*endly.Context) RequestBuilder
	setRequest(*Request) RequestBuilder
	build() (*http.Request, error)
}
type requestBuilder struct {
	context *endly.Context
	request *Request
}

func newRequestBuilder() RequestBuilder {
	return &requestBuilder{}
}
func (b *requestBuilder) setContext(c *endly.Context) RequestBuilder {
	b.context = c
	return b
}
func (b *requestBuilder) setRequest(r *Request) RequestBuilder {
	b.request = r
	return b
}
func (b *requestBuilder) build() (*http.Request, error) {
	var err error

	//Expand request
	b.request = b.request.Expand(b.context)

	//Build request body
	requestBody, isBase64Encoded, err := b.buildRequestBody()
	if err != nil {
		return nil, err
	}

	//Create http request
	httpRequest, err := http.NewRequest(strings.ToUpper(b.request.Method), b.request.URL, requestBody)
	if err != nil {
		return nil, err
	}

	//Set transfer encoding
	if isBase64Encoded {
		setTransferEncoding(httpRequest, "")
	}

	//Initialize http request with headers and cookies from SendRequest
	copyHeaders(b.request.Header, httpRequest.Header)
	addCookies(httpRequest.Header, b.request.Cookies)

	//Add session cookies
	b.buildSessionCookies(httpRequest)

	return httpRequest, nil
}
func (b *requestBuilder) buildRequestBody() (io.Reader, bool, error) {
	var isBase64Encoded bool
	var body []byte
	var err error
	var reader io.Reader
	var ok bool
	if len(b.request.Body) > 0 {
		body = []byte(b.request.Body)
		if b.request.RequestUdf != "" {
			transformed, err := endly.TransformWithUDF(b.context, b.request.RequestUdf, b.request.URL, string(body))
			if err != nil {
				return nil, isBase64Encoded, err
			}
			if body, ok = transformed.([]byte); !ok {
				body = []byte(toolbox.AsString(transformed))
			}
		}
		isBase64Encoded = strings.HasPrefix(string(body), "base64:")
		body, err = endly.FromPayload(string(body))
		if err != nil {
			return nil, isBase64Encoded, err
		}
		reader = bytes.NewReader(body)
	}

	return reader, isBase64Encoded, nil
}
func (b *requestBuilder) buildSessionCookies(r *http.Request) {
	sessionCookies := b.context.State().GetMap("cookies")
	if sessionCookies != nil && len(sessionCookies) > 0 {
		contextCookies := make([]*http.Cookie, len(sessionCookies))
		for _, v := range sessionCookies {
			if cookie, ok := v.(*http.Cookie); ok {
				contextCookies = append(contextCookies, cookie)
			}
		}
		addCookies(r.Header, contextCookies)
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
func addCookies(h http.Header, c Cookies) {
	if c == nil || len(c) == 0 {
		return
	}
	for _, cookie := range c {
		if v := cookie.String(); v != "" {
			h.Add("Cookie", v)
		}
	}
}

func setTransferEncoding(r *http.Request, encoding string) {
	if r.TransferEncoding == nil {
		r.TransferEncoding = make([]string, 0)
	}
	r.TransferEncoding = append(r.TransferEncoding, encoding)
}
