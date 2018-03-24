package model

import (
	"github.com/viant/endly"
)

//represents pipelines
type Pipelines []*Pipeline

//Pipeline represents sequential workflow/action execution.
type Pipeline struct {
	Name      string                 `description:"pipeline task name"`
	Workflow  string                 `description:"workflow (URL[:tasks]) selector "`
	Action    string                 `description:"action (service.action) selector "`
	Params    map[string]interface{} `description:"workflow or action parameters"`
	When      string                 `description:"run criteria"`
	Pipelines Pipelines              `description:"workflow or action subsequent pipelines"`
}



//Select selects pipelines matching supplied selector
func (p *Pipelines) Select(selector TasksSelector) Pipelines {
	if selector.RunAll() {
		return *p
	}
	var result = make([]*Pipeline, 0)
	for _, task := range selector.Tasks() {
		for _, pipeline := range *p {
			if task == pipeline.Name {
				result = append(result, pipeline)
				continue
			}
			if len(pipeline.Pipelines) > 0 {
				selected := pipeline.Pipelines.Select(selector)
				result = append(result, selected...)
			}
		}
	}
	return result
}

//NewActivity returns pipline activity
func (p *Pipeline) NewActivity(context *endly.Context) *Activity {
	var action = &Action{
		NeatlyTag:&NeatlyTag{Tag:p.Name},
		ServiceRequest:&ServiceRequest{},
		Repeater: &Repeater{},
	}
	if p.Action != "" {
		selector := ActionSelector(p.Action)
		action.Service = selector.Service()
		action.Action = selector.Action()
	} else if p.Workflow != "" {
		action.Service = "workflow"
		action.Action = "run"
	}
	action.Request = p.Params
	var state = context.State()
	return  NewActivity(context, action, state)
}