package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
	"io"
	ioutil "io/ioutil"
	"net/http"
	"strings"
)

func initializeContext(c *endly.Context) {
	var state = c.State()
	if !state.Has("cookies") {
		state.Put("cookies", data.NewMap())
	}
	if !state.Has(TripsKey) {
		state.Put(TripsKey, newTrips())
	}
}

//copyHeaders copy source to target headers
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

//copyExpandedHeaders copy and expand headers
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

//readBody reads and transform response body if needed
func readBody(httpResponse *http.Response, response *Response, expectBinary bool) {
	body, err := ioutil.ReadAll(httpResponse.Body)
	defer httpResponse.Body.Close()
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		return
	}
	if expectBinary { //readBody reads http body, if request send a binary format, expect binary format back
		buf := new(bytes.Buffer)
		encoder := base64.NewEncoder(base64.StdEncoding, buf)
		_, _ = encoder.Write(body)
		_ = encoder.Close()
		response.Body = "base64:" + string(buf.Bytes())

	} else {
		response.Body = string(body)
	}
}

//getRequestBodyReader returns request body reader
func getRequestBodyReader(request *http.Request, repeat int) (func() io.ReadCloser, error) {
	if repeat == 0 || request.ContentLength == 0 {
		return func() io.ReadCloser {
			return request.Body
		}, nil
	}

	bodyCache, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	return func() io.ReadCloser {
		return ioutil.NopCloser(bytes.NewReader(bodyCache))
	}, nil
}

func replaceResponseBodyIfNeeded(sendHTTPRequest *Request, responseBody string) string {
	if len(sendHTTPRequest.Replace) > 0 {
		for k, v := range sendHTTPRequest.Replace {
			responseBody = strings.Replace(responseBody, k, v, len(responseBody))
		}
	}
	return responseBody
}
