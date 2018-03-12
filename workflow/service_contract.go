package workflow

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

//WorkflowParams represents workflow parameters
type WorkflowParams map[string]interface{}

//RunRequest represents workflow run request
type RunRequest struct {
	EnableLogging     bool            `description:"flag to enable logging"`
	LoggingDirectory  string          `description:"log directory"`
	WorkflowURL       string          `description:"workflow URL if workflow is not found in the registry, it is loaded"`
	Name              string          `required:"true" description:"name defined in workflow document"`
	Params            WorkflowParams  `description:"workflow parameters, accessibly by paras.[Key], if PublishParameters is set, all parameters are place in context.state"`
	Tasks             string          `required:"true" description:"coma separated task list or '*'to run all tasks sequencialy"` //tasks to run with coma separated list or '*', or empty string for all tasks
	TagIDs            string          `description:"coma separated TagID list, if present in a task, only matched runs, other task run as normal"`
	PublishParameters bool            `description:"flag to publish parameters directly into context state"`
	Async             bool            `description:"flag to run it asynchronously. Do not set it your self runner sets the flag for the first workflow"`
	EventFilter       map[string]bool `description:"optional CLI filter option,key is either package name or package name.request/event prefix "`
}

//RunResponse represents workflow run response
type RunResponse struct {
	Data      map[string]interface{} //  data populated by  .Post variable section.
	SessionID string                 //session id
}

//WorkflowSelector represents an expression to invoke workflow with all or specified task:  URL[:tasks]
type WorkflowSelector string

//Pipeline represents a workflow with parameters to run
type Pipeline struct {
	Key      WorkflowSelector `description:"an expression to invoke workflow with all or specified task:  URL[:tasks]"`
	Value    WorkflowParams   `description:"workflow parameters"`
	resource *url.Resource    //workflow resource
}

//NewPipeline creates a new pipeline
func NewPipeline(selector string, params map[string]interface{}) *Pipeline {
	return &Pipeline{
		Key:   WorkflowSelector(selector),
		Value: WorkflowParams(params),
	}
}

//PipelineRequest represent request to run one or more workflow.
type PipelineRequest struct {
	Namespace string         `description:"if specified  it represents prefix for all relative workflow URL"`
	Async     bool           `default:"true" description:"flag to run pipeline asynchronously"`
	Params    WorkflowParams `description:"shared parameters to be included if not present in pipline level parameters"`
	Pipeline  []*Pipeline    `required:"true" description:"workflows with parameters to run"`
}

//NewPipelineRequest returns new pipeline request
func NewPipelineRequest(namespace string, async bool, params WorkflowParams, pipeline ...*Pipeline) *PipelineRequest {
	return &PipelineRequest{
		Namespace: namespace,
		Async:     async,
		Params:    params,
		Pipeline:  pipeline,
	}
}

//Response represent a pipeline response.
type PipelineResponse struct {
	Response map[string]*RunResponse
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
	Task                 string      `description:"task to run if matched"`
	Value                interface{} `required:"true" description:"matching sourceKey value"`
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
