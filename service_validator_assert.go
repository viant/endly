package endly

import "github.com/viant/assertly"

//ValidatorAssertRequest represent assert request
type ValidatorAssertRequest struct {
	TagID       string
	Name        string
	Description string
	Actual      interface{} `required:"true" description:"actual value/data structure"`
	Expected    interface{} `required:"true" description:"expected value/data structure"`
}

//ValidatorAssertResponse represent validation response
type ValidatorAssertResponse struct {
	*assertly.Validation
}
