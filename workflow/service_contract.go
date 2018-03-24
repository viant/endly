package workflow

import (
	"errors"
	"github.com/viant/endly/model"
	"github.com/viant/endly/msg"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"strings"
)

//RunRequest represents workflow runWorkflow request
type RunRequest struct {
	EnableLogging     bool                   `description:"flag to enable logging"`
	LogDirectory      string                 `description:"log directory"`
	EventFilter       map[string]bool        `description:"optional CLI filter option,key is either package name or package name.request/event prefix "`
	Async             bool                   `description:"flag to runWorkflow it asynchronously. Do not set it your self runner sets the flag for the first workflow"`
	Params            map[string]interface{} `description:"workflow parameters, accessibly by paras.[Key], if PublishParameters is set, all parameters are place in context.state"`
	PublishParameters bool                   `default:"true" description:"flag to publish parameters directly into context state"`
	URL               string                 `description:"workflow URL if workflow is not found in the registry, it is loaded"`
	Name              string                 `required:"true" description:"name defined in workflow document"`
	TagIDs            string                 `description:"coma separated TagID list, if present in a task, only matched runs, other task runWorkflow as normal"`
	Tasks             string                 `required:"true" description:"coma separated task list, if empty or '*' runs all tasks sequencialy"` //tasks to runWorkflow with coma separated list or '*', or empty string for all tasks
	*InlinePipeline
}

//HasPipeline returns true if request has inline piplines
func (r *RunRequest) HasPipeline() bool {
	if r.InlinePipeline == nil {
		return false
	}
	return len(r.Pipelines) > 0 || len(r.Pipeline) > 0
}

//Init initialises request
func (r *RunRequest) Init() (err error) {
	r.Params, err = util.NormalizeMap(r.Params, true)
	if err != nil {
		return err
	}
	if r.HasPipeline() {
		return r.InlinePipeline.Init(r)
	}

	if r.URL == "" {
		r.URL = r.Name
	}
	if r.URL != "" {
		r.URL = model.WorkflowSelector(r.URL).URL()
	}
	if r.Name == "" {
		r.Name = model.WorkflowSelector(r.URL).Name()
	} else {
		if index := strings.LastIndex(r.Name, "/"); index != -1 {
			r.Name = string(r.Name[index+1:])
		}
	}

	return nil
}

//Validate checks if request is valid
func (r *RunRequest) Validate() error {
	if r.HasPipeline() {
		return nil
	}
	if r.Name == "" {
		return errors.New("name was empty")
	}
	if r.URL == "" {
		return errors.New("url was empty")
	}
	return nil
}

//NewRunRequest creates a new runWorkflow request
func NewRunRequest(workflow string, params map[string]interface{}, publishParams bool) *RunRequest {
	selector := model.WorkflowSelector(workflow)
	return &RunRequest{
		Params:            params,
		PublishParameters: publishParams,
		URL:               selector.URL(),
		Name:              selector.Name(),
		Tasks:             selector.Tasks(),
	}
}

//NewRunRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	var request = &RunRequest{}
	var resource = url.NewResource(URL)
	return request, resource.Decode(request)
}

//RunResponse represents workflow runWorkflow response
type RunResponse struct {
	Data      map[string]interface{} //  data populated by  .Post variable section.
	SessionID string                 //session id
}

//MapEntry represents a workflow with parameters to runWorkflow
type MapEntry struct {
	Key   string      `description:"preserved order map entry key"`
	Value interface{} `description:"preserved order map entry value"`
}

//InlinePipeline represent sequence of workflow/action to runWorkflow
type InlinePipeline struct {
	Pipeline  []*MapEntry     `required:"true" description:"key value representing Pipelines in simplified form"`
	Pipelines model.Pipelines `description:"actual Pipelines (derived from Pipeline)"`
}

func (r *InlinePipeline) toPipeline(source interface{}, pipeline *model.Pipeline, runRequest *RunRequest) (err error) {
	var aMap map[string]interface{}

	if aMap, err = util.NormalizeMap(source, false); err != nil {
		return err
	}
	if workflow, ok := aMap[pipelineWorkflow]; ok {
		pipeline.Workflow = toolbox.AsString(workflow)
		delete(aMap, pipelineWorkflow)

		pipeline.Params = aMap
		pipeline.Params, _ = util.NormalizeMap(pipeline.Params, true)
		util.Append(runRequest.Params, pipeline.Params, false)
		return nil
	}
	if action, ok := aMap[pipelineAction]; ok {
		pipeline.Action = toolbox.AsString(action)
		delete(aMap, pipelineAction)
		pipeline.Params = aMap
		pipeline.Params, _ = util.NormalizeMap(pipeline.Params, true)
		util.Append(runRequest.Params, pipeline.Params, false)
		return nil
	}

	if e := toolbox.ProcessMap(source, func(key, value interface{}) bool {
		subPipeline := &model.Pipeline{
			Name:      toolbox.AsString(key),
			Pipelines: make([]*model.Pipeline, 0),
		}
		if err = r.toPipeline(value, subPipeline, runRequest); err != nil {
			return false
		}
		pipeline.Pipelines = append(pipeline.Pipelines, subPipeline)
		return true
	}); e != nil {
		return e
	}
	return err
}

//Init initialises
func (r *InlinePipeline) Init(runRequest *RunRequest) (err error) {
	if len(r.Pipelines) > 0 {
		return nil
	}
	runRequest.Params, _ = util.NormalizeMap(runRequest.Params, true)
	r.Pipelines = make([]*model.Pipeline, 0)
	for _, entry := range r.Pipeline {
		pipeline := &model.Pipeline{
			Name:      entry.Key,
			Pipelines: make([]*model.Pipeline, 0),
		}
		if err := r.toPipeline(entry.Value, pipeline, runRequest); err != nil {
			return err
		}
		r.Pipelines = append(r.Pipelines, pipeline)
	}

	selector := model.TasksSelector(runRequest.Tasks)
	if ! selector.RunAll() {
		r.Pipelines = r.Pipelines.Select(selector)
	}
	return nil
}


//RegisterRequest represents workflow register request
type RegisterRequest struct {
	*model.Workflow
}

//RegisterResponse represents workflow register response
type RegisterResponse struct {
	Source *url.Resource
}

// LoadRequest represents workflow load request from the specified source
type LoadRequest struct {
	Source *url.Resource
}

// LoadResponse represents loaded workflow
type LoadResponse struct {
	*model.Workflow
}

// SwitchCase represent matching candidate case
type SwitchCase struct {
	*model.ServiceRequest `description:"action to runWorkflow if matched"`
	Task  string          `description:"task to runWorkflow if matched"`
	Value interface{}     `required:"true" description:"matching sourceKey value"`
}

// SwitchRequest represent switch action request
type SwitchRequest struct {
	SourceKey string        `required:"true" description:"sourceKey for matching value"`
	Cases     []*SwitchCase `required:"true" description:"matching value cases"`
	Default   *SwitchCase   `description:"in case no value was match case"`
}

//Match matches source with supplied action request.
func (r *SwitchRequest) Match(source interface{}) *SwitchCase {
	for _, switchCase := range r.Cases {
		if switchCase.Value == source {
			return switchCase
		}
	}
	return r.Default
}

// SwitchResponse represents actual action or task response
type SwitchResponse interface{}

//Validate checks if workflow is valid
func (r *SwitchRequest) Validate() error {
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	if len(r.Cases) == 0 {
		return errors.New("cases were empty")
	}
	for _, matchingCase := range r.Cases {
		if matchingCase.Value == nil {
			return errors.New("cases.value was empty")
		}
	}
	return nil
}

// GotoRequest represents goto task action, this request will terminate current task execution to switch to specified task
type GotoRequest struct {
	Task string
}

// GotoResponse represents workflow task response
type GotoResponse interface{}

// ExitRequest represents workflow exit request, to exit a caller workflow
type ExitRequest struct {
	Source *url.Resource
}

// ExitResponse represents workflow exit response
type ExitResponse struct{}

// FailRequest represents fail request
type FailRequest struct {
	Message string
}

// FailResponse represents workflow exit response
type FailResponse struct{}

//NopRequest represent no operation
type NopRequest struct{}

//NopParrotRequest represent parrot request
type NopParrotRequest struct {
	In interface{}
}

//PrintRequest represent print request
type PrintRequest struct {
	Message string
	Style   int
	Error   string
}

//Messages returns messages
func (r *PrintRequest) Messages() []*msg.Message {

	var result = msg.NewMessage(nil, nil)
	if r.Message != "" {
		result.Items = append(result.Items, msg.NewStyled(r.Message, r.Style))
	}
	if r.Error != "" {
		result.Items = append(result.Items, msg.NewStyled(r.Message, msg.MessageStyleError))
	}
	return []*msg.Message{result}
}
