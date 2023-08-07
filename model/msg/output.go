package msg

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type OutputEvent struct {
	Message string
	Tag     string
	Value   interface{}
}

func (e *OutputEvent) Messages() []*Message {
	info := ""
	if content, err := yaml.Marshal(e.Value); err == nil {
		info = string(content)
	}
	return []*Message{
		NewMessage(NewStyled(e.Message, MessageStyleGeneric),
			NewStyled(e.Tag, MessageStyleGeneric),
			NewStyled(info, MessageStyleOutput),
		),
	}
}

func NewOutputEvent(message string, tag string, value interface{}) *OutputEvent {
	return &OutputEvent{
		Message: message,
		Tag:     tag,
		Value:   value,
	}
}

// StdoutEvent represents an execution event end
type StdoutEvent struct {
	Info   string
	Stdout string
	Error  string
}

// Messages returns messages
func (e *StdoutEvent) Messages() []*Message {
	return []*Message{
		NewMessage(NewStyled(fmt.Sprintf("%v", e.Info), MessageStyleGeneric), NewStyled("stdout", MessageStyleGeneric),
			NewStyled(e.Stdout, MessageStyleOutput)),
	}
}

// NewStdoutEvent crates a new execution start event value
func NewStdoutEvent(info string, stdout string) *StdoutEvent {
	return &StdoutEvent{
		Info:   info,
		Stdout: stdout,
	}
}
