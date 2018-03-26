package workflow

import (
	"github.com/viant/endly/model"
	"github.com/viant/toolbox/data"
)

//LoadedEvent represents workflow load event
type LoadedEvent struct {
	Workflow *model.Workflow
}

//NewLoadedEvent create a new workflow load event.
func NewLoadedEvent(workflow *model.Workflow) *LoadedEvent {
	return &LoadedEvent{Workflow: workflow}
}

//InitEvent represents a new workflow init event
type InitEvent struct {
	Tasks string
	State map[string]interface{}
}

//NewInitEvent creates a new workflow init event.
func NewInitEvent(tasks string, state data.Map) *InitEvent {
	return &InitEvent{
		Tasks: tasks,
		State: state.AsEncodableMap(),
	}
}

//EndEvent represents Activity end event type.
type EndEvent struct {
	SessionID string
}

//NewEndEvent create a new EndEvent
func NewEndEvent(sessionID string) *EndEvent {
	return &EndEvent{
		SessionID: sessionID,
	}
}

//AsyncEvent represents an async action event.
type AsyncEvent struct {
	ServiceAction *model.Action
}

//NewAsyncEvent creates a new AsyncEvent.
func NewAsyncEvent(action *model.Action) *AsyncEvent {
	return &AsyncEvent{action}
}

//PipelineEvent represents a pipeline event
type PipelineEvent struct {
	Name string
}

//
////Messages returns messages
//func (e *PipelineEvent) Messages() []*msg.Message {
//	return []*msg.Message{
//		msg.NewMessage(msg.NewStyled(fmt.Sprintf("%s", e.Name), msg.MessageStyleGroup), msg.NewStyled("pipeline", msg.MessageStyleGeneric)),
//	}
//}


//NewPipelineEvent creates a new pipeline event
func NewPipelineEvent(pipeline *model.Pipeline) *PipelineEvent {
	return &PipelineEvent{
		Name: pipeline.Name,
	}
}



//ModifiedStateEvent represent modified state event
type ModifiedStateEvent struct {
	Variables model.Variables
	In        map[string]interface{}
	Modified  map[string]interface{}
}

//NewModifiedStateEvent creates a new modified state event.
func NewModifiedStateEvent(variables model.Variables, in, out data.Map) *ModifiedStateEvent {
	var result = &ModifiedStateEvent{
		Variables: variables,
		In:        make(map[string]interface{}),
		Modified:  make(map[string]interface{}),
	}

	for _, variable := range variables {
		from := data.ExtractPath(variable.From)
		result.In[from], _ = in.GetValue(from)

		name := data.ExtractPath(variable.Name)

		result.Modified[name], _ = out.GetValue(name)
	}
	return result
}