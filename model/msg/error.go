package msg

import (
	"fmt"
	"github.com/viant/toolbox"
)

func ReportError(err error) error {
	fileName, funcName, line := toolbox.CallerInfo(4)
	return fmt.Errorf("%v at %v:%v -> %v", err, fileName, line, funcName)
}

// ErrorEvent represents a Sleep
type ErrorEvent struct {
	Error string
}

// Messages returns messages
func (e *ErrorEvent) Messages() []*Message {

	error := NewStyled(fmt.Sprintf("%v\n", e.Error), MessageStyleError)
	return []*Message{
		NewMessage(NewStyled("", MessageStyleError), NewStyled("error", MessageStyleError), error)}
}

// NewErrorEvent creates a new error event
func NewErrorEvent(message string) *ErrorEvent {
	return &ErrorEvent{
		Error: message,
	}
}
