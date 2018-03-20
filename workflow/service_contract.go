package workflow

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

//Params represents parameters
type Params map[string]interface{}

//AbstractRun represents a base run request
type AbstractRun struct {
	EnableLogging bool            `description:"flag to enable logging"`
	LogDirectory  string          `description:"log directory"`
	EventFilter   map[string]bool `description:"optional CLI filter option,key is either package name or package name.request/event prefix "`
	Async         bool            `description:"flag to run it asynchronously. Do not set it your self runner sets the flag for the first workflow"`
	Params        Params          `description:"workflow parameters, accessibly by paras.[Key], if PublishParameters is set, all parameters are place in context.state"`
}

//RunRequest represents workflow run request
type RunRequest struct {
	*AbstractRun
	URL               string `description:"workflow URL if workflow is not found in the registry, it is loaded"`
	Name              string `required:"true" description:"name defined in workflow document"`
	Tasks             string `required:"true" description:"coma separated task list or '*'to run all tasks sequencialy"` //tasks to run with coma separated list or '*', or empty string for all tasks
	TagIDs            string `description:"coma separated TagID list, if present in a task, only matched runs, other task run as normal"`
	PublishParameters bool   `description:"flag to publish parameters directly into context state"`
}

//Init initialises request
func (r *RunRequest) Init() error {
	if r.AbstractRun == nil {
		r.AbstractRun = &AbstractRun{}
	}
	if r.URL == "" {
		r.URL = r.Name
	}
	if r.URL != "" {
		r.URL = WorkflowSelector(r.URL).URL()
	}
	if r.Name == "" {
		r.Name = WorkflowSelector(r.URL).Name()
	} else {
		if index := strings.LastIndex(r.Name, "/");index != -1 {
			r.Name = string(r.Name[index+1:])
		}
	}
	return nil
}

//Validate checks if request is valid
func (r *RunRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name was empty")
	}
	if r.URL == "" {
		return errors.New("url was empty")
	}
	return nil
}

//NewRunRequest creates a new run request
func NewRunRequest(selector WorkflowSelector, params map[string]interface{}, publishParams bool) *RunRequest {
	return &RunRequest{
		AbstractRun: &AbstractRun{
			Params: params,
		},
		URL:               selector.URL(),
		Name:              selector.Name(),
		Tasks:             selector.Tasks(),
		PublishParameters: publishParams,
	}
}

//NewRunRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	var request = &RunRequest{}
	var resource = url.NewResource(URL)
	return request, resource.Decode(request)
}

//RunResponse represents workflow run response
type RunResponse struct {
	Data      map[string]interface{} //  data populated by  .Post variable section.
	SessionID string                 //session id
}

//WorkflowSelector represents an expression to invoke workflow with all or specified task:  URL[:tasks]
type WorkflowSelector string

//ActionSelector represents an expression to invoke endly action:  service:Action
type ActionSelector string

//MapEntry represents a workflow with parameters to run
type MapEntry struct {
	Key   string
	Value interface{}
}

type Pipeline struct {
	Name      string
	Skip      bool
	Workflow  WorkflowSelector
	Action    ActionSelector
	Params    Params
	Pipelines []*Pipeline
}

//NewPipeline creates a new pipeline
func NewPipeline(key string, value interface{}) *MapEntry {
	return &MapEntry{
		Key:   key,
		Value: value,
	}
}

//PipeRequest represent request to run workflows/actions sequentialy if previous was completed without error
type PipeRequest struct {
	*AbstractRun
	Pipeline  []*MapEntry `required:"true" description:"key value representing pipelines in simplified form"`
	Pipelines []*Pipeline `description:"actual pipelines (derived from Pipeline)"`
}

func (r *PipeRequest) toPipeline(source interface{}, pipeline *Pipeline) (err error) {
	var aMap map[string]interface{}
	if aMap, err = util.NormalizeMap(source); err != nil {
		return err
	}
	if workflow, ok := aMap[pipelineWorkflow]; ok {
		pipeline.Workflow = WorkflowSelector(toolbox.AsString(workflow))
		delete(aMap, pipelineWorkflow)
		pipeline.Params = Params(aMap)
		pipeline.Params.AppendParams(r.Params, false)
		return nil
	}
	if workflow, ok := aMap[pipelineAction]; ok {
		pipeline.Action = ActionSelector(toolbox.AsString(workflow))
		delete(aMap, pipelineAction)
		pipeline.Params = Params(aMap)
		pipeline.Params.AppendParams(r.Params, false)
		return nil
	}
	if e := toolbox.ProcessMap(source, func(key, value interface{}) bool {
		subPipeline := &Pipeline{
			Name:      toolbox.AsString(key),
			Pipelines: make([]*Pipeline, 0),
		}
		if err = r.toPipeline(value, subPipeline); err != nil {
			return false
		}
		pipeline.Pipelines = append(pipeline.Pipelines, subPipeline)
		return true
	}); e != nil {
		return e
	}
	return err
}

func (r *PipeRequest) Init() (err error) {
	if r.AbstractRun == nil {
		r.AbstractRun = &AbstractRun{}
	}
	r.Params, err = util.NormalizeMap(r.Params)
	if err != nil {
		return err
	}
	if len(r.Pipelines) > 0 {
		return nil
	}
	r.Pipelines = make([]*Pipeline, 0)
	for _, entry := range r.Pipeline {
		pipeline := &Pipeline{
			Name:      entry.Key,
			Pipelines: make([]*Pipeline, 0),
		}
		if err := r.toPipeline(entry.Value, pipeline); err != nil {
			return err
		}
		r.Pipelines = append(r.Pipelines, pipeline)
	}
	return nil
}

//NewPipelineRequest returns new pipeline request
func NewPipelineRequest(async bool, params Params, pipeline ...*MapEntry) *PipeRequest {
	return &PipeRequest{
		AbstractRun: &AbstractRun{
			Async:  async,
			Params: params,
		},
		Pipeline: pipeline,
	}
}

//NewPipelineRequestFromURL creates a new pipeline request from URL
func NewPipelineRequestFromURL(URL string) (*PipeRequest, error) {
	resource := url.NewResource(URL)
	var response = &PipeRequest{}
	return response, resource.Decode(response)
}

//Response represent a pipeline response.
type PipeResponse struct {
	Response map[string]interface{}
}

//RegisterRequest represents workflow register request
type RegisterRequest struct {
	*endly.Workflow
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
	*endly.Workflow
}

// SwitchCase represent matching candidate case
type SwitchCase struct {
	*endly.ActionRequest `description:"action to run if matched"`

	Task  string      `description:"task to run if matched"`
	Value interface{} `required:"true" description:"matching sourceKey value"`
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
func (r *PrintRequest) Messages() []*endly.Message {

	var result = endly.NewMessage(nil, nil)
	if r.Message != "" {
		result.Items = append(result.Items, endly.NewStyledText(r.Message, r.Style))
	}
	if r.Error != "" {
		result.Items = append(result.Items, endly.NewStyledText(r.Message, endly.MessageStyleError))
	}
	return []*endly.Message{result}
}

//URL returns workflow url
func (s WorkflowSelector) URL() string {
	URL, _, _ := s.split()
	return URL
}

func (s WorkflowSelector) IsRelative() bool {
	URL := s.URL()
	if strings.Contains(URL, "://") || strings.HasPrefix(URL, "/") {
		return false
	}
	return true
}

//split returns selector URL, name and tasks
func (s WorkflowSelector) split() (URL, name, tasks string) {
	var sel = string(s)
	protoPosition := strings.LastIndex(sel, "://")
	taskPosition := strings.LastIndex(sel, ":")
	if protoPosition != -1 {
		taskPosition = -1
		selWithoutProto := string(sel[protoPosition+3:])
		if position := strings.LastIndex(selWithoutProto, ":"); position != -1 {
			taskPosition = protoPosition + 3 + position
		}
	}
	URL = sel
	tasks = "*"
	if taskPosition != -1 {
		tasks = string(URL[taskPosition+1:])
		URL = string(URL[:taskPosition])

	}
	var ext = path.Ext(URL)
	if ext == "" {
		_, name = path.Split(URL)
		URL += ".csv"
	} else {
		_, name = path.Split(string(URL[:len(URL)-len(ext)]))
	}
	return URL, name, tasks
}

//Name returns selector workflow name
func (s WorkflowSelector) Name() string {
	_, name, _ := s.split()
	return name
}

//Tasks returns selector tasks
func (s WorkflowSelector) Tasks() string {
	_, _, tasks := s.split()
	return tasks

}

//Action returns action
func (s ActionSelector) Action() string {
	pair := strings.Split(string(s), ":")
	if len(pair) == 2 {
		return pair[1]
	}
	return ""
}

//Service returns service
func (s ActionSelector) Service() string {
	pair := strings.Split(string(s), ":")
	if len(pair) == 2 {
		return pair[0]
	}
	return string(s)
}

//AppendMap source to dest map
func (p *Params) AppendParams(source Params, override bool) {
	for k, v := range source {
		if _, ok := (*p)[k]; ok && !override {
			continue
		}
		(*p)[k] = v
	}
}

//GetAbstractRun returns base request for supplied request or error
func GetAbstractRun(request interface{}) (*AbstractRun, error) {
	switch req := request.(type) {
	case *RunRequest:
		if req == nil {
			return nil, fmt.Errorf("request %T was nil", request)
		}
		return req.AbstractRun, nil
	case *PipeRequest:
		if req == nil {
			return nil, fmt.Errorf("request %T  was nil", request)
		}
		return req.AbstractRun, nil
	}
	return nil, fmt.Errorf("unsupported tyep %T, exacted %T or %T", request, &RunRequest{}, &PipeRequest{})
}
