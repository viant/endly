package workflow

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

//WorkflowLoadedEvent represents workflow load event
type WorkflowLoadedEvent struct {
	Workflow *endly.Workflow
}

//NewWorkflowLoadedEvent create a new workflow load event.
func NewWorkflowLoadedEvent(workflow *endly.Workflow) *WorkflowLoadedEvent {
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
	ServiceAction *endly.ServiceAction
}

//NewWorkflowAsyncEvent creates a new WorkflowAsyncEvent.
func NewWorkflowAsyncEvent(action *endly.ServiceAction) *WorkflowAsyncEvent {
	return &WorkflowAsyncEvent{action}
}

//PipelineEvent represents a pipeline event
type PipelineEvent struct {
	Name string
}

//Messages returns messages
func (e *PipelineEvent) Messages() []*endly.Message {
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("PIPELINE: %s", e.Name), endly.MessageStyleGroup), endly.NewStyledText("pipe", endly.MessageStyleGeneric)),
	}
}

func NewPipelineEvent(pipeline *Pipeline) *PipelineEvent {
	return &PipelineEvent{
		Name: pipeline.Name,
	}
}
