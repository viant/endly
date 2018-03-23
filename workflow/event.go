package workflow

import (
	"fmt"
	"github.com/viant/toolbox/data"
	"github.com/viant/endly/model"
	"github.com/viant/endly/msg"
)

//WorkflowLoadedEvent represents workflow load event
type WorkflowLoadedEvent struct {
	Workflow *model.Workflow
}

//NewWorkflowLoadedEvent create a new workflow load event.
func NewWorkflowLoadedEvent(workflow *model.Workflow) *WorkflowLoadedEvent {
	return &WorkflowLoadedEvent{Workflow: workflow}
}

//WorkflowInitEvent represents a new workflow init event
type WorkflowInitEvent struct {
	Tasks string
	State map[string]interface{}
}

//NewWorkflowInitEvent creates a new workflow init event.
func NewWorkflowInitEvent(tasks string, state data.Map) *WorkflowInitEvent {
	return &WorkflowInitEvent{
		Tasks: tasks,
		State: state.AsEncodableMap(),
	}
}

//WorkflowEndEvent represents Activity end event type.
type WorkflowEndEvent struct {
	SessionID string
}

//NewWorkflowEndEvent create a new WorkflowEndEvent
func NewWorkflowEndEvent(sessionID string) *WorkflowEndEvent {
	return &WorkflowEndEvent{
		SessionID: sessionID,
	}
}

//WorkflowAsyncEvent represents an async action event.
type WorkflowAsyncEvent struct {
	ServiceAction *model.Action
}

//NewWorkflowAsyncEvent creates a new WorkflowAsyncEvent.
func NewWorkflowAsyncEvent(action *model.Action) *WorkflowAsyncEvent {
	return &WorkflowAsyncEvent{action}
}

//PipelineEvent represents a pipeline event
type PipelineEvent struct {
	Name string
}

//Messages returns messages
func (e *PipelineEvent) Messages() []*msg.Message {
	return []*msg.Message{
		msg.NewMessage(msg.NewStyledText(fmt.Sprintf("PIPELINE: %s", e.Name), msg.MessageStyleGroup), msg.NewStyledText("pipe", msg.MessageStyleGeneric)),
	}
}

func NewPipelineEvent(pipeline *model.Pipeline) *PipelineEvent {
	return &PipelineEvent{
		Name: pipeline.Name,
	}
}
