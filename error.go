package endly

import (
	"fmt"
	"strings"
)

//Error repreents an exception
type Error struct {
	Path  []string
	error error
}

//Unshift appends at the  begining service, action
func (e *Error) Unshift(service, action string) {
	e.Path = append([]string{fmt.Sprintf("%v.%v", service, action)}, e.Path...)
}

//Error returns en error
func (e *Error) Error() string {
	return fmt.Sprintf("%v at %v", e.error, strings.Join(e.Path, "/"))
}

//NewAbstractException returns new abstract exception
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
