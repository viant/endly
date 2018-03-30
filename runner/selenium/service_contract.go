package selenium

import (
	"errors"
	"fmt"
	"github.com/viant/endly/model"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"github.com/viant/endly/testing/validator"
)

//StartRequest represents a selenium server start request
type StartRequest struct {
	Target     *url.Resource
	Port       int
	Sdk        string
	SdkVersion string
	Version    string
}

//NewStartRequestFromURL creates a new start request from URL
func NewStartRequestFromURL(URL string) (*StartRequest, error) {
	var result = &StartRequest{}
	var resource = url.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

//StartResponse repreents a selenium server stop request
type StartResponse struct {
	Pid             int
	ServerPath      string
	GeckodriverPath string
}

//StopRequest represents server stop request
type StopRequest struct {
	Target *url.Resource
	Port   int
}

//NewStopRequestFromURL creates a new start request from URL
func NewStopRequestFromURL(URL string) (*StopRequest, error) {
	var result = &StopRequest{}
	var resource = url.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

//StopResponse represents a selenium stop request
type StopResponse struct {
}

//OpenSessionResponse represents open session response.
type OpenSessionResponse struct {
	SessionID string
}

//CloseSessionRequest represents close session request.
type CloseSessionRequest struct {
	SessionID string
}

//NewCloseSessionRequestFromURL creates a new close session request from URL
func NewCloseSessionRequestFromURL(URL string) (*CloseSessionRequest, error) {
	var result = &CloseSessionRequest{}
	var resource = url.NewResource(URL)
	err := resource.Decode(result)
	return result, err
}

//CloseSessionResponse represents close session response.
type CloseSessionResponse struct {
	SessionID string
}

//WebDriverCallRequest represents selenium call driver request
type WebDriverCallRequest struct {
	SessionID string
	Call      *MethodCall
}

//ServiceCallResponse represents selenium call response
type ServiceCallResponse struct {
	Result []interface{}
	Data   data.Map
}

//WebElementSelector represents a web element selector
type WebElementSelector struct {
	By    string //selector type
	Value string //selector value
	Key   string //optional result key
}

//WebElementCallRequest represents a web element call reqesut
type WebElementCallRequest struct {
	SessionID string
	Selector  *WebElementSelector
	Call      *MethodCall
}

//WebElementCallResponse represents seleniun web element response
type WebElementCallResponse struct {
	Result      []interface{}
	LookupError string
	Data        map[string]interface{}
}

//RunRequest represents group of selenium web elements calls
type RunRequest struct {
	SessionID      string
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
	Actions        []*Action
	Commands       []interface{} `description:"list of selenium command: {web element selector}.WebElementMethod(params),  or WebDriverMethod(params), or wait map "`
	Expect       interface{}   `description:"If specified it will validated response as actual"`
}

func (r *RunRequest) asWaitAction(parser *parser, candidate interface{}) (*Action, error) {
	if aMap, err := util.NormalizeMap(candidate, true); err == nil {
		command, ok := aMap["command"]
		if !ok {
			return nil, fmt.Errorf("command was missing: %v", candidate)
		}
		action, err := parser.Parse(toolbox.AsString(command))
		if err != nil {
			return nil, err
		}
		action.Calls[0].Wait = &model.Repeater{}
		err = toolbox.DefaultConverter.AssignConverted(action.Calls[0].Wait, aMap)
		return action, err
	}
	return nil, fmt.Errorf("sunupported command: %T", candidate)
}

func (r *RunRequest) Init() error {
	if len(r.Actions) > 0 {
		for _, action := range r.Actions {
			if action.Selector != nil {
				_ = action.Selector.Init()
				if action.Selector.Key == "" {
					action.Selector.Key = action.Key
				}
				if action.Selector.Key == "" {
					action.Selector.Key = action.Selector.Value
				}
			}
		}
		return nil
	}
	if len(r.Commands) == 0 {
		return nil
	}

	r.Actions = make([]*Action, 0)
	var previousAction *Action
	parser := &parser{}
	for _, candidate := range r.Commands {

		command, ok := candidate.(string)
		if !ok {
			action, err := r.asWaitAction(parser, candidate)
			if err != nil {
				return err
			}
			r.Actions = append(r.Actions, action)
			continue
		}
		action, err := parser.Parse(command)
		if err != nil {
			return fmt.Errorf("invalid command: %v, %v", command, err)
		}
		if previousAction != nil {
			if previousAction.Selector != nil && action.Selector != nil && previousAction.Selector.Value == action.Selector.Value {
				previousAction.Calls = append(previousAction.Calls, action.Calls[0])
				continue
			}
		}
		r.Actions = append(r.Actions, action)
		previousAction = action
	}
	return nil
}

//NewRunRequest creates a new run request
func NewRunRequest(sessionID, browser string, remote *url.Resource, actions ...*Action) *RunRequest {
	return &RunRequest{
		SessionID:      sessionID,
		Browser:        browser,
		RemoteSelenium: remote,
		Actions:        actions,
	}
}

//NewRunRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	resource := url.NewResource(URL)
	var result = &RunRequest{}
	return result, resource.Decode(result)
}

//RunResponse represents selenium call response
type RunResponse struct {
	SessionID    string
	Data         map[string]interface{}
	LookupErrors []string
	Assert *validator.AssertResponse
}

//MethodCall represents selenium call.
type MethodCall struct {
	Wait       *model.Repeater
	Method     string
	Parameters []interface{}
}

//Action represents various calls on web element
type Action struct {
	Key      string //optional result key
	Selector *WebElementSelector
	Calls    []*MethodCall
}

//NewAction creates a new action
func NewAction(key, selector string, method string, params ...interface{}) *Action {
	var result = &Action{
		Key: key,
		Calls: []*MethodCall{
			{
				Method:     method,
				Parameters: params,
			},
		},
	}
	if selector != "" {
		var webSelector = WebSelector(selector)
		result.Selector = &WebElementSelector{}
		result.Selector.By, result.Selector.Value = webSelector.ByAndValue()
		result.Selector.Key = result.Key
	}
	return result
}

//Validate validates run request.
func (r *RunRequest) Validate() error {
	if r.SessionID == "" {
		if r.RemoteSelenium == nil {
			return fmt.Errorf("both SessionID and Remote were empty")
		}
		if r.Browser == "" {
			return fmt.Errorf("both SessionID and Browser were empty")
		}
	}

	if len(r.Actions) == 0 {
		return fmt.Errorf("both actions/commands were empty")
	}
	return nil
}

//NewMethodCall creates a new method call
func NewMethodCall(method string, repeatable *model.Repeater, parameters ...interface{}) *MethodCall {
	return &MethodCall{
		Wait:       repeatable,
		Method:     method,
		Parameters: parameters,
	}
}

//OpenSessionRequest represents open session request
type OpenSessionRequest struct {
	Browser        string
	RemoteSelenium *url.Resource `description:"http selenium server endpoint"`
	SessionID      string        `description:"if specified this ID will be used for a sessionID"`
}

//Validate validate open session request
func (r *OpenSessionRequest) Validate() error {
	if r.RemoteSelenium == nil {
		return errors.New("remote (remote selenium endpoint) was empty")
	}
	if r.RemoteSelenium.URL == "" {
		return errors.New("remote.URL (selenium resource URL) was empty")
	}
	if r.Browser == "" {
		return errors.New("browser was empty")
	}
	return nil
}

//NewOpenSessionRequest creates a new open session request
func NewOpenSessionRequest(browser string, remote *url.Resource) *OpenSessionRequest {
	return &OpenSessionRequest{
		Browser:        browser,
		RemoteSelenium: remote,
	}
}
