package lambda

import (
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
	"time"
)

type EventTriggerInfo struct {
	// The identifier of the event source mapping.
	UUID *string `yaml:"uuid,omitempty" type:"string"`

	// The maximum number of items to retrieve in a single batch.
	BatchSize *int64 `yaml:"batchSize,omitempty" min:"1" type:"integer"`

	// The Amazon Resource Name (ARN) of the event source.
	EventSourceArn *string `yaml:"sourceARN,omitempty" type:"string"`

	// The date that the event source mapping was last updated, in Unix time seconds.
	LastModified *time.Time `yaml:"modified,omitempty" type:"timestamp"`

	// The result of the last AWS Lambda invocation of your Lambda function.
	LastProcessingResult *string `yaml:"lastProcessingResult,omitempty" type:"string"`

	// The state of the event source mapping. It can be one of the following: Creating,
	// Enabling, Enabled, Disabling, Disabled, Updating, or Deleting.
	State *string `yaml:"state,omitempty" type:"string"`

	// The cause of the last state change, either User initiated or Lambda initiated.
	StateTransitionReason *string `yaml:"stateTransitionReason,omitempty" type:"string"`
}


type SetupFunctionEvent struct {
	Function     *FunctionInfo
	Triggers     []*EventTriggerInfo `yaml:"triggers,omitempty"`
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
		Triggers: make([]*EventTriggerInfo, 0),
	}
	if len(output.EventMappings) > 0 {
		for _, mapping := range output.EventMappings {
			result.Triggers = append(result.Triggers, &EventTriggerInfo{
				UUID:                  mapping.UUID,
				BatchSize:             mapping.BatchSize,
				EventSourceArn:        mapping.EventSourceArn,
				LastModified:          mapping.LastModified,
				LastProcessingResult:  mapping.LastProcessingResult,
				StateTransitionReason: mapping.StateTransitionReason,
			})
		}
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
