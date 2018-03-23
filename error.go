package endly

import (
	"fmt"
	"strings"
)

//Error represents an workflow execution error
type Error struct {
	Path  []string
	error error
}

//Unshift appends supplied pathFragments at the beginning
func (e *Error) Unshift(pathFragments ...string) {
	var pathFragment = strings.Join(pathFragments, ".")
	e.Path = append([]string{pathFragment}, e.Path...)
}

//Error returns en error
func (e *Error) Error() string {
	return fmt.Sprintf("%v at %v", e.error, strings.Join(e.Path, "/"))
}

//NewError returns new workflow exception or update path
func NewError(service, action string, err error) error {
	if abstractException, ok := err.(*Error); ok {
		abstractException.Unshift(service, action)
		return abstractException
	}
	return &Error{
		Path:  []string{fmt.Sprintf("%v.%v", service, action)},
		error: err,
	}
}
