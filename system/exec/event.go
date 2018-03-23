package exec

import (
	"fmt"
	"github.com/viant/endly/msg"
)

//StdinEvent represents an execution event start
type StdinEvent struct {
	SessionID string
	Stdin     string
}

//Messages returns messages
func (e *StdinEvent) Messages() []*msg.Message {
	return []*msg.Message{
		msg.NewMessage(msg.NewStyledText(fmt.Sprintf("%v", e.SessionID), msg.MessageStyleGeneric), msg.NewStyledText("stdin", msg.MessageStyleGeneric),
			msg.NewStyledText(e.Stdin, msg.MessageStyleInput)),
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
func (e *StdoutEvent) Messages() []*msg.Message {
	return []*msg.Message{
		msg.NewMessage(msg.NewStyledText(fmt.Sprintf("%v", e.SessionID), msg.MessageStyleGeneric), msg.NewStyledText("stdout", msg.MessageStyleGeneric),
			msg.NewStyledText(e.Stdout, msg.MessageStyleOutput)),
	}
}

//NewStdoutEvent crates a new execution start event value
func NewStdoutEvent(sessionID string, stdout string, err error) *StdoutEvent {
	return &StdoutEvent{
		SessionID: sessionID,
		Stdout:    stdout,
	}
}
