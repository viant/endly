package workflow

import (
	"errors"
	"fmt"
	"github.com/viant/endly/model/location"
	"path"
	"strings"

	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
)

// RunRequest represents workflow runWorkflow request
type RunRequest struct {
	EnableLogging     bool                   `description:"flag to enable logging"`
	LogDirectory      string                 `description:"log directory"`
	FailureCount      int                    `description:"max number of failures CLI reported per validation"`
	SummaryFormat     string                 `description:"summary format: xml|json|yaml, summary file is not produced if this is empty"`
	EventFilter       map[string]bool        `description:"optional CLI filter option,key is either package name or package name.request/event prefix "`
	Async             bool                   `description:"flag to runWorkflow it asynchronously. Do not set it your self runner sets the flag for the first workflow"`
	Params            map[string]interface{} `description:"workflow parameters, accessibly by paras.[Key], if PublishParameters is set, all parameters are place in context.state"`
	PublishParameters bool                   `default:"true" description:"flag to publish parameters directly into context state"`
	SharedState       bool                   `description:"by default workflow uses a separate cloned context copy, if this is flag context will be shared with a caller workflow state"`
	URL               string                 `description:"workflow URL if workflow is not found in the registry, it is loaded"`
	Name              string                 `required:"true" description:"name defined in workflow document"`
	StateKey          string                 `description:"if specified workflow params and data will be visible globally with this key, default is inherited from workflow name"`
	Source            *location.Resource     `description:"run request location "`
	AssetURL          string
	TagIDs            string `description:"coma separated TagID list, if present in a task, only matched runs, other task runWorkflow as normal"`
	Tasks             string `required:"true" description:"coma separated task list, if empty or '*' runs all tasks sequentially"` //tasks to runWorkflow with coma separated list or '*', or empty string for all tasks
	Interactive       bool
	*model.InlineWorkflow
	workflow *model.Workflow //inline workflow from pipeline
}

// Init initialises request
func (r *RunRequest) Init() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%v (%v)", err, r.AssetURL)
		}
	}()

	if r.Params, err = util.NormalizeMap(r.Params, true); err != nil {
		return err
	}
	if r.Tasks == "" || r.Tasks == "$tasks" {
		r.Tasks = "*"
	}

	if r.InlineWorkflow != nil && (len(r.InlineWorkflow.Pipeline) > 0) {
		if r.AssetURL == "" {
			return fmt.Errorf("asset URL is required for inline workflow")
		}
		name := r.Name
		baseURL, URI := toolbox.URLSplit(r.AssetURL)
		if name == "" && r.Name != "" {
			name = strings.Replace(URI, path.Ext(URI), "", 1)
		}
		r.workflow, err = r.InlineWorkflow.AsWorkflow(name, baseURL)
		if err != nil {
			return err
		}
		r.workflow.Source = location.NewResource(r.AssetURL)
		if r.StateKey == "" {
			r.StateKey = r.Name
		}
		return r.workflow.Init()
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
	if r.StateKey == "" {
		r.StateKey = r.Name
	}
	return nil
}

// Validate checks if request is valid
func (r *RunRequest) Validate() error {
	if r.workflow != nil {
		return r.workflow.Validate()
	}

	if r.Name == "" {
		return errors.New("name was empty")
	}
	if r.URL == "" {
		return errors.New("url was empty")
	}
	return nil
}

// NewRunRequest creates a new runWorkflow request
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

// NewRunRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	var request = &RunRequest{}
	var resource = location.NewResource(URL)
	return request, resource.Decode(request)
}

// RunResponse represents workflow runWorkflow response
type RunResponse struct {
	Data      map[string]interface{} //  data populated by  .Post variable section.
	SessionID string                 //session id
}

// RegisterRequest represents workflow register request
type RegisterRequest struct {
	*model.Workflow
}

// RegisterResponse represents workflow register response
type RegisterResponse struct {
	Source *location.Resource
}

// LoadRequest represents workflow load request from the specified source
type LoadRequest struct {
	Source *location.Resource
}

// Validate checks if request is valid
func (r *LoadRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}

// LoadResponse represents loaded workflow
type LoadResponse struct {
	*model.Workflow
}

// SwitchCase represent matching candidate case
type SwitchCase struct {
	*model.ServiceRequest `description:"action to runWorkflow if matched"`
	Task                  string      `description:"task to runWorkflow if matched"`
	Value                 interface{} `required:"true" description:"matching sourceKey value"`
}

// SwitchRequest represent switch action request
type SwitchRequest struct {
	SourceKey string        `required:"true" description:"sourceKey for matching value"`
	Cases     []*SwitchCase `required:"true" description:"matching value cases"`
	Default   *SwitchCase   `description:"in case no value was match case"`
}

// Match matches source with supplied action request.
func (r *SwitchRequest) Match(source interface{}) *SwitchCase {
	for _, switchCase := range r.Cases {
		if toolbox.AsString(switchCase.Value) == toolbox.AsString(source) {
			return switchCase
		}
	}
	return r.Default
}

// SwitchResponse represents actual action or task response
type SwitchResponse interface{}

// Validate checks if workflow is valid
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
	Source *location.Resource
}

// ExitResponse represents workflow exit response
type ExitResponse struct{}

// FailRequest represents fail request
type FailRequest struct {
	Message string
}

// FailResponse represents workflow exit response
type FailResponse struct{}

// NopRequest represent no operation
type NopRequest struct{}

// NopParrotRequest represent parrot request
type NopParrotRequest struct {
	In interface{}
}

// SetEnvRequest represents set env request
type SetEnvRequest struct {
	Env map[string]string `description:"dynamically change current run endly os environment variables"`
}

// SetEnvResponse returns original env setup
type SetEnvResponse struct {
	Env map[string]string
}

// PrintRequest represent print request
type PrintRequest struct {
	Message string
	Style   int
	Error   string
}

// Messages returns messages
func (r *PrintRequest) Messages() []*msg.Message {

	var result = msg.NewMessage(nil, nil)
	if r.Message != "" {
		result.Items = append(result.Items, msg.NewStyled(r.Message, r.Style))
	}
	if r.Error != "" {
		result.Items = append(result.Items, msg.NewStyled(r.Error, msg.MessageStyleError))
	}
	return []*msg.Message{result}
}
