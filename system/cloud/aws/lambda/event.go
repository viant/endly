package lambda

import (
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
)

type SetupFunctionEvent struct {
	Function *FunctionInfo
}

func (e *SetupFunctionEvent) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(e.Function.Name, msg.MessageStyleGeneric),
			msg.NewStyled("function", msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
		),
	}
}

type FunctionInfo struct {
	Name string
	Arn  string
}

func NewSetupFunctionEvent(output *SetupFunctionOutput) *SetupFunctionEvent {
	var result = &SetupFunctionEvent{
		Function: &FunctionInfo{
			Name: *output.FunctionName,
			Arn:  *output.FunctionArn,
		},
	}
	return result
}

func (i *SetupFunctionOutput) Messages() []*msg.Message {
	if i == nil || i.FunctionConfiguration == nil || i.RoleInfo == nil {
		return nil
	}
	event := NewSetupFunctionEvent(i)
	return event.Messages()

}
