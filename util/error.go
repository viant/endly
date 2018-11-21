package util

import (
	"strings"
)

//NotSuchResourceError represents generic no such resource error
type NotSuchResourceError struct{ Resource string }

func (e *NotSuchResourceError) Error() string {
	return "no such resource " + e.Resource
}

//NewNotSuchResourceError create new NewNotSuchResourceError
func NewNotSuchResourceError(resource string) *NotSuchResourceError {
	return &NotSuchResourceError{Resource: resource}
}

//ClassifyErrorIfMatched classify error with concrete implementation if needed
func ClassifyErrorIfMatched(err error) error {
	if strings.Contains(err.Error(), "no such file or directory") {
		return NewNotSuchResourceError(strings.Replace(err.Error(), "no such file or directory", "", 1))
	}
	return err
}

//IsNotSuchResourceError returns trus if error is of NotSuchResourceError type
func IsNotSuchResourceError(err error) bool {
	_, ok := err.(*NotSuchResourceError)
	return ok
}
