package validator

import (
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
)

// AssertRequest represent assert request
type AssertRequest struct {
	TagID            string
	Name             string
	Description      string
	Actual           interface{} `required:"true" description:"actual value/data structure"`
	Expect           interface{} `required:"true" description:"expected value/data structure"`
	Expected         interface{} //Deprecated
	Source           interface{} //optional validation source
	Ignore           interface{}
	OmitEmpty        bool
	NormalizeKVPairs bool //flag to normalize kv pairs into map if possible (i.e, when using yaml)
}

func (r *AssertRequest) IgnoreKeys() []interface{} {
	var keys = make([]interface{}, 0)
	switch actual := r.Ignore.(type) {
	case []interface{}:
		for i := range actual {
			keys = append(keys, actual[i])
		}
	case map[string]interface{}:
		for k := range actual {
			keys = append(keys, k)
		}
	case map[interface{}]interface{}:
		for k := range actual {
			keys = append(keys, k)
		}
	}
	return keys
}

// AssertResponse represent validation response
type AssertResponse struct {
	*assertly.Validation
}

func (r *AssertRequest) Init() error {
	if r.Expect == nil {
		r.Expect = r.Expected
	}

	if r.Expect == nil {
		return nil
	}

	if r.NormalizeKVPairs {
		if normalized, err := toolbox.NormalizeKVPairs(r.Expect); err == nil {
			r.Expect = normalized
		}
	}
	return nil
}

// Assertion returns validation slice
func (r *AssertResponse) Assertion() []*assertly.Validation {
	if r == nil {
		return []*assertly.Validation{}
	}
	return []*assertly.Validation{r.Validation}
}

// NewAssertRequest creates a new assertRequest
func NewAssertRequest(tagID string, name string, description string, source, expected, actual interface{}) *AssertRequest {
	return &AssertRequest{
		Source:      source,
		TagID:       tagID,
		Name:        name,
		Description: description,
		Expected:    expected,
		Actual:      actual,
	}
}
