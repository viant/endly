package rest

import "github.com/viant/endly/testing/validator"

//Request represents a send request
type Request struct {
	URL     string
	Method  string
	Request interface{}
	Expect interface{} `description:"If specified it will validated response as actual"`
}

//Response represents a rest response
type Response struct {
	Response interface{}
	Assert *validator.AssertResponse
}
