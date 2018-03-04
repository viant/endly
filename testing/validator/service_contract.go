package validator

import "github.com/viant/assertly"

//AssertRequest represent assert request
type AssertRequest struct {
	TagID       string
	Name        string
	Description string
	Actual      interface{} `required:"true" description:"actual value/data structure"`
	Expected    interface{} `required:"true" description:"expected value/data structure"`
}

//AssertResponse represent validation response
type AssertResponse struct {
	*assertly.Validation
}
