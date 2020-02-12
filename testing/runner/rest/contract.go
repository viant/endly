package rest

import (
	"github.com/viant/endly/model"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
)

//Request represents a send request
type Request struct {
	Options     map[string]interface{} `description:"http client options_: key value pairs, where key is one of the following: HTTP options_:RequestTimeoutMs,TimeoutMs,KeepAliveTimeMs,TLSHandshakeTimeoutMs,ResponseHeaderTimeoutMs,MaxIdleConns,FollowRedirects"`
	httpOptions []*toolbox.HttpOptions
	*model.Repeater
	URL     string
	Method  string
	Request interface{}
	Expect  interface{} `description:"If specified it will validated response as actual"`
}

func (r *Request) Init() error {
	r.httpOptions = make([]*toolbox.HttpOptions, 0)
	if len(r.Options) > 0 {
		for k, v := range r.Options {
			r.httpOptions = append(r.httpOptions, &toolbox.HttpOptions{Key: k, Value: v})
		}
	}
	return nil
}

//Response represents a rest response
type Response struct {
	Response interface{}
	Assert   *validator.AssertResponse
}
