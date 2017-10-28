package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

//FailedTest represents a failed test
type FailedTest struct {
	PathIndex int
	Path      string
	Expected  interface{}
	Actual    interface{}
	Message   string
}

//ExtractPathIndex extract index  from path
func ExtractPathIndex(path string) (int, string) {
	startPathPosition := strings.Index(path, "]")
	if startPathPosition == -1 || startPathPosition > 20 {

		return -1, path
	}
	pathSegments := strings.Split(path, "]")
	if len(pathSegments) == 0 {
		return -1, path
	}
	indexSegment := pathSegments[0]
	endPositionOfTheFirstPathSegment := len(indexSegment) + 1
	startPathPosition = strings.LastIndex(indexSegment, "[")
	if startPathPosition == -1 {
		fmt.Printf("3\n")
		return -1, path
	}
	var candidate = string(indexSegment[startPathPosition+1:])
	var index = toolbox.AsInt(candidate)
	if index > 0 || strings.Contains(candidate, toolbox.AsString(index)) {
		if len(path) == endPositionOfTheFirstPathSegment-1 {
			return index, ""
		}
		return index, string(path[endPositionOfTheFirstPathSegment:])
	}
	return -1, path
}

//NewFailedTest creates a new FailedTest instance
func NewFailedTest(path, message string, expected, actual interface{}) *FailedTest {
	var result = &FailedTest{
		Message:  message,
		Expected: expected,
		Actual:   actual,
	}
	result.PathIndex, result.Path = ExtractPathIndex(path)

	return result
}

//ValidationInfo represents assertion info
type ValidationInfo struct {
	Name        string
	Tag         string
	TagIndex    string
	TestPassed  int
	FailedTests []*FailedTest
}

//AddFailure appends failure
func (ar *ValidationInfo) AddFailure(failedTest *FailedTest) {
	if len(ar.FailedTests) == 0 {
		ar.FailedTests = make([]*FailedTest, 0)
	}
	ar.FailedTests = append(ar.FailedTests, failedTest)
}

//HasFailure returns true if at least one failure has been registered
func (ar *ValidationInfo) HasFailure() bool {
	return len(ar.FailedTests) > 0
}

//Message provides summary message
func (ar *ValidationInfo) Message() string {

	var failed = make([]string, 0)

	for i := 0; i < len(ar.FailedTests); i++ {
		failed = append(failed, ar.FailedTests[i].Message)
	}

	return fmt.Sprintf("Passed: %v\nFailed:%v\n-----\n\t%v\n",
		ar.TestPassed,
		len(ar.FailedTests),
		strings.Join(failed, "\n\t"),
	)
}
