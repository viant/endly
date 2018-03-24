package cli

import "github.com/viant/assertly"

//Asserted represent validation response
type Asserted interface {
	Assertion() []*assertly.Validation
}
