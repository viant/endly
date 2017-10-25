package endly

import (
	"fmt"
	"strings"
)

//AssertionInfo represents assertion info
type AssertionInfo struct {
	TestPassed int
	TestFailed []string
}

//AddFailure appends failure
func (ar *AssertionInfo) AddFailure(message string) {
	if len(ar.TestFailed) == 0 {
		ar.TestFailed = make([]string, 0)
	}
	ar.TestFailed = append(ar.TestFailed, message)
}

//HasFailure returns true if at least one failure has been registered
func (ar *AssertionInfo) HasFailure() bool {
	return len(ar.TestFailed) > 0
}

//Message provides summary message
func (ar *AssertionInfo) Message() string {
	return fmt.Sprintf("Passed: %v\nFailed:%v\n-----\n\t%v\n",
		ar.TestPassed,
		len(ar.TestFailed),
		strings.Join(ar.TestFailed, "\n\t"),
	)
}
