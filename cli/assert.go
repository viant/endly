package cli

import "github.com/viant/assertly"

//Assertable represent validation response
type Assertable interface {
	Assertion() []*assertly.Validation
}
