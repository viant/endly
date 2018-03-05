package exec

import (
	"fmt"
	"github.com/viant/endly"
)

var executionStartFilter = endly.NewFilteredReporter("exec", "stdin", "exec.run")

//ExecutionStartEvent represents an execution event start
type ExecutionStartEvent struct {
	SessionID string
	Stdin     string
}

//CanReport returns true if filter has matching event key enabled reporting option
func (e *ExecutionStartEvent) CanReport(filter map[string]bool) bool {
	return executionStartFilter.CanReport(filter)
}

//Messages returns messages
func (e *ExecutionStartEvent) Messages() []*endly.Message {
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v", e.SessionID), endly.MessageStyleGeneric), endly.NewStyledText("stdin", endly.MessageStyleGeneric),
			false,
			endly.NewStyledText(e.Stdin, endly.MessageStyleInput)),
	}
}

//NewExecutionStartEvent crates a new execution start event value
func NewExecutionStartEvent(sessionID string, stdin string) *ExecutionStartEvent {
	return &ExecutionStartEvent{
		SessionID: sessionID,
		Stdin:     stdin,
	}
}

var executionEndFilter = endly.NewFilteredReporter("exec", "stdout", "exec.run")

//ExecutionEndEvent represents an execution event end
type ExecutionEndEvent struct {
	SessionID string
	Stdout    string
	Error     string
}

//CanReport returns true if filter has matching event key enabled reporting option
func (e *ExecutionEndEvent) CanReport(filter map[string]bool) bool {
	return executionEndFilter.CanReport(filter)
}

//Messages returns messages
func (e *ExecutionEndEvent) Messages() []*endly.Message {
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v", e.SessionID), endly.MessageStyleGeneric), endly.NewStyledText("stdout", endly.MessageStyleGeneric),
			false,
			endly.NewStyledText(e.Stdout, endly.MessageStyleOutput)),
	}
}

//NewExecutionEndEvent crates a new execution start event value
func NewExecutionEndEvent(sessionID string, stdout string, err error) *ExecutionEndEvent {
	return &ExecutionEndEvent{
		SessionID: sessionID,
		Stdout:    stdout,
	}
}
