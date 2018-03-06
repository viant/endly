package exec

import (
	"fmt"
	"github.com/viant/endly"
)

//StdinEvent represents an execution event start
type StdinEvent struct {
	SessionID string
	Stdin     string
}

//Messages returns messages
func (e *StdinEvent) Messages() []*endly.Message {
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v", e.SessionID), endly.MessageStyleGeneric), endly.NewStyledText("stdin", endly.MessageStyleGeneric),
			endly.NewStyledText(e.Stdin, endly.MessageStyleInput)),
	}
}

//NewSdtinEvent crates a new execution start event value
func NewSdtinEvent(sessionID string, stdin string) *StdinEvent {
	return &StdinEvent{
		SessionID: sessionID,
		Stdin:     stdin,
	}
}

//StdoutEvent represents an execution event end
type StdoutEvent struct {
	SessionID string
	Stdout    string
	Error     string
}

//Messages returns messages
func (e *StdoutEvent) Messages() []*endly.Message {
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v", e.SessionID), endly.MessageStyleGeneric), endly.NewStyledText("stdout", endly.MessageStyleGeneric),
			endly.NewStyledText(e.Stdout, endly.MessageStyleOutput)),
	}
}

//NewStdoutEvent crates a new execution start event value
func NewStdoutEvent(sessionID string, stdout string, err error) *StdoutEvent {
	return &StdoutEvent{
		SessionID: sessionID,
		Stdout:    stdout,
	}
}
