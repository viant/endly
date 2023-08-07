package http

import (
	"github.com/viant/endly"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"net/http"
)

// Response represents Http response
type Response struct {
	//ServiceRequest     *ServiceRequest
	Code        int
	Header      http.Header
	Cookies     map[string]*http.Cookie
	Body        string
	JSONBody    interface{} `description:"structure data if Body was JSON"`
	TimeTakenMs int
	Error       string
}

func (r *Response) TransformBodyIfNeeded(context *endly.Context, request *Request) error {
	if request.ResponseUdf == "" {
		return nil
	}
	transformed, err := udf.TransformWithUDF(context, request.ResponseUdf, request.URL, r.Body)
	if err != nil {
		return err
	}
	if toolbox.IsMap(transformed) {
		r.Body, _ = toolbox.AsJSONText(transformed)
	} else {
		r.Body = toolbox.AsString(transformed)
	}
	r.Body = replaceResponseBodyIfNeeded(request, r.Body)
	return nil
}

func (r *Response) UpdateCookies(target data.Map) {
	for k, cookie := range r.Cookies {
		target.Put(k, cookie.Value)
	}
}

// Merge merge response from HTTP response
func (r *Response) Merge(httpResponse *http.Response, expectBinary bool) {
	r.Code = httpResponse.StatusCode
	r.Header = make(map[string][]string)
	copyHeaders(httpResponse.Header, r.Header)
	readBody(httpResponse, r, expectBinary)
	var responseCookies Cookies = httpResponse.Cookies()
	r.Cookies = responseCookies.IndexByName()
}

// NewResponse creates a new response
func NewResponse() *Response {
	var response = &Response{}
	response.Header = make(map[string][]string)
	return response
}
