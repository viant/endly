package gcp

import (
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
)

type OutputEvent struct {
	Message string
	Tag     string
	Value   interface{}
}

func (e *OutputEvent) Messages() []*msg.Message {
	info := ""

	if e.Value != nil && toolbox.IsMap(e.Value) && toolbox.IsStruct(e.Value) {
		aMap := toolbox.AsMap(e.Value)
		e.Value = toolbox.DeleteEmptyKeys(aMap)
	}

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
