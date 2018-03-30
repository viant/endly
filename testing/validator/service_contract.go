package validator

import "github.com/viant/assertly"

//AssertRequest represent assert request
type AssertRequest struct {
	TagID       string
	Name        string
	Description string
	Actual      interface{} `required:"true" description:"actual value/data structure"`
	Expected    interface{} `required:"true" description:"expected value/data structure"`
	Source interface{} 		//optional validation source
}

//AssertResponse represent validation response
type AssertResponse struct {
	*assertly.Validation
}

//Assertion returns validation slice
func (r *AssertResponse) Assertion() []*assertly.Validation {
	if r == nil {
		return []*assertly.Validation{}
	}
	return []*assertly.Validation{r.Validation}
}

//NewAssertRequest creates a new assertRequest
func NewAssertRequest(tagID string, name string, description string, source, expected, actual interface{}) *AssertRequest{
	return &AssertRequest{
		Source:source,
		TagID:tagID,
		Name:name,
		Description:description,
		Expected:expected,
		Actual:actual,
	}
}