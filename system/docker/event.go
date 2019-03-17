package docker

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
)

type OutputEvent struct {
	Message string
	Tag     string
	Value   interface{}
}

func (e *OutputEvent) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e.Value); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(e.Message, msg.MessageStyleGeneric),
			msg.NewStyled(e.Tag, msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
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

//StdoutEvent represents an execution event end
type StdoutEvent struct {
	Info   string
	Stdout string
	Error  string
}

//Messages returns messages
func (e *StdoutEvent) Messages() []*msg.Message {
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(fmt.Sprintf("%v", e.Info), msg.MessageStyleGeneric), msg.NewStyled("stdout", msg.MessageStyleGeneric),
			msg.NewStyled(e.Stdout, msg.MessageStyleOutput)),
	}
}

//NewStdoutEvent crates a new execution start event value
func NewStdoutEvent(info string, stdout string) *StdoutEvent {
	return &StdoutEvent{
		Info:   info,
		Stdout: stdout,
	}
}
