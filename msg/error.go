package msg

import (
	"github.com/viant/toolbox"
	"fmt"
)


func ReportError(err error) error {
	fileName, funcName, line := toolbox.CallerInfo(4)
	return fmt.Errorf("%v at %v:%v -> %v", err, fileName, line, funcName)
}


//ErrorEvent represents a Sleep
type ErrorEvent struct {
	Error string
}

//Messages returns messages
func (e *ErrorEvent) Messages() []*Message {
	return []*Message{
		NewMessage(NewStyledText(fmt.Sprintf("%v", e.Error), MessageStyleError), NewStyledText("error", MessageStyleError))}
}

//NewErrorEvent creates a new error event
func NewErrorEvent(message string) *ErrorEvent {
	return &ErrorEvent{
		Error: message,
	}
}
