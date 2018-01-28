package endly

import "github.com/viant/assertly"

//ValidatorAssertRequest represent assert request
type ValidatorAssertRequest struct {
	TagID       string
	Description string
	Actual      interface{} //actual data structure
	Expected    interface{} //expecte data structure
}


//ValidatorAssertResponse represent validation response
type ValidatorAssertResponse struct {
	*assertly.Validation
}