package model

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

const(
	//CatchPipelineTask  represent a task name that execute if error occured and defined
	CatchPipelineTask = "catch"
	//DeferPipelineTask represent a task name that always execute if defined
	DeferPipelineTask = "defer"
)

//Pipelines represents pipelines
type Pipelines []*Pipeline

//RunnableCount returns number of pipeline that run either action or workflow
func (p *Pipelines) RunnableCount() int {
	var result = 0
	for _, pipeline := range *p {
		if pipeline.Workflow != "" || pipeline.Action != "" {
			result++
			continue
		}
		if len(pipeline.Pipelines) > 0 {
			result += pipeline.Pipelines.RunnableCount()
		}
	}
	return result
}

//Pipeline represents sequential workflow/action execution.
type Pipeline struct {
	Name        string                 `description:"pipeline task name"`
	Workflow    string                 `description:"workflow (URL[:tasks]) selector "`
	Action      string                 `description:"action (service.action) selector "`
	Params      map[string]interface{} `description:"workflow or action parameters"`
	Description string                 `description:"description"`
	Request     interface{}            `description:"external action request location, otherwise params are used to form request"`
	Pipelines   Pipelines              `description:"workflow or action subsequent pipelines"`
	When        string                 `description:"run criteria"`
	Init        interface{}            `description:"state initalization "`
	Post        interface{}            `description:"post execution state update "`
}

//Select selects pipelines matching supplied selector
func (p *Pipelines) Select(selector TasksSelector) Pipelines {
	if selector.RunAll() {
		return *p
	}

	var allowed = make(map[string]bool)
	allowed[CatchPipelineTask] = true
	allowed[DeferPipelineTask]= true
	for _, task := range selector.Tasks() {
		allowed[task] = true
	}


	var result Pipelines = []*Pipeline{}
	for _, pipeline := range *p {
		if len(pipeline.Pipelines) > 0  {
			if allowed[pipeline.Name] {
				result = append(result, pipeline.Pipelines...)
			} else {
				var selected = pipeline.Pipelines.Select(selector)
				if len(selected) > 0 {
					result = append(result, selected...)
				}
			}
		}

		if allowed[pipeline.Name] {
			result = append(result, pipeline)
		}
	}
	return result
}



func (p *Pipeline) initRequestIfNeeded(baseURL string) (err error) {
	if p.Request == nil {
		return nil
	}
	if toolbox.IsString(p.Request) {
		var requestMap = make(map[string]interface{})
		err := util.DecodeMap(baseURL, toolbox.AsString(p.Request), requestMap)
		if err != nil {
			return err
		}
		var state = data.Map(p.Params)
		p.Request = state.Expand(requestMap)
		return err
	}
	return nil
}

//NewActivity returns pipeline activity
func (p *Pipeline) NewActivity(context *endly.Context) *Activity {
	var action = &Action{
		NeatlyTag: &NeatlyTag{Tag: p.Name},
		ServiceRequest: &ServiceRequest{
			Description: p.Description,
		},
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
	if p.Request != nil {
		action.Request = p.Request
	} else {
		action.Request = p.Params
	}
	var state = context.State()
	return NewActivity(context, action, state)
}

//MapEntry represents pipeline parameters or attributes if key has '@' prefix.
type MapEntry struct {
	Key   string      `description:"preserved order map entry key"`
	Value interface{} `description:"preserved order map entry value"`
}

//Inline represent sequence of workflow/action to run, map entry represent bridge between YAML, JSON and actual domain model Pipelines abstraction.
type Inline struct {
	Pipeline  []*MapEntry            `required:"true" description:"key value representing Pipelines in simplified form"`
	Pipelines Pipelines              `description:"actual Pipelines (derived from Pipeline)"`
	Defaults  map[string]interface{} `description:"default value for pipline parameters"`
	Init      interface{}            `description:"init state expression"`
	Post      interface{}            `description:"post processing update state expression"`
}

func (p Inline) split(source interface{}) (attributes, params map[string]interface{}, err error) {
	aMap, err := util.NormalizeMap(source, false)
	attributes = make(map[string]interface{})
	params = make(map[string]interface{})
	for k, v := range aMap {
		if strings.HasPrefix(k, "@") {
			attributes[string(k[1:])] = v
			continue
		}
		attributes[k] = v
		params[k] = v
	}
	return attributes, params, err
}

func (p *Inline) toPipeline(baseURL string, source interface{}, name string, defaultParams map[string]interface{}) (pipeline *Pipeline, err error) {
	attributes, params, err := p.split(source)
	if err != nil {
		return nil, err
	}
	pipeline = &Pipeline{}
	if err = toolbox.DefaultConverter.AssignConverted(pipeline, attributes); err != nil {
		return nil, err
	}

	pipeline.Params, _ = util.NormalizeMap(params, true)
	util.Append(pipeline.Params, defaultParams, false)

	if err = pipeline.initRequestIfNeeded(baseURL); err != nil {
		return nil, err
	}

	if pipeline.Init != nil {
		pipeline.Init, err = GetVariables(baseURL, pipeline.Init)
	}
	if pipeline.Post != nil {
		pipeline.Post, err = GetVariables(baseURL, pipeline.Post)
	}
	pipeline.Name = name
	if pipeline.Workflow != "" || pipeline.Action != "" {
		return pipeline, nil
	}

	pipeline.Pipelines = make([]*Pipeline, 0)
	var nextPipeline *Pipeline

	if e := toolbox.ProcessMap(source, func(key, value interface{}) bool {
		if !toolbox.IsSlice(value) {
			return true
		}
		nextPipeline, err = p.toPipeline(baseURL, value, toolbox.AsString(key), defaultParams)
		if err != nil {
			return false
		}
		pipeline.Pipelines = append(pipeline.Pipelines, nextPipeline)
		return true
	}); e != nil {
		return nil, e
	}
	return pipeline, err
}

//Init initialises inline pipeline
func (p *Inline) InitTasks(baseURL string, selector TasksSelector, defaultParams map[string]interface{}) (err error) {
	if len(p.Pipelines) > 0 {
		return nil
	}
	p.Init, err = GetVariables(baseURL, p.Init)
	p.Post, err = GetVariables(baseURL, p.Post)
	p.Pipelines = make([]*Pipeline, 0)
	for _, entry := range p.Pipeline {
		pipeline, err := p.toPipeline(baseURL, entry.Value, entry.Key, defaultParams)
		if err != nil {
			return err
		}
		p.Pipelines = append(p.Pipelines, pipeline)
	}

	if !selector.RunAll() {
		p.Pipelines = p.Pipelines.Select(selector)
	}

	if len(p.Pipelines) == 0 || p.Pipelines.RunnableCount() == 0 {
		return fmt.Errorf("no pipelines matched with tasks selector: '%v'", string(selector))
	}
	return nil
}
