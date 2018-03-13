package selenium

import (
	"errors"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"strings"
)

//StartRequest represents a selenium server start request
type StartRequest struct {
	Target     *url.Resource
	Port       int
	Sdk        string
	SdkVersion string
	Version    string
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
	Result    []interface{}
	Extracted map[string]string
}

//WebElementSelector represents a web element selector
type WebElementSelector struct {
	By    string //selector type
	Value string //selector value
	Key   string //optional result key, otherwise value is used
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
}

//Validate checks is selector is valid.
func (s *WebElementSelector) Validate() error {
	if s.Value == "" {
		return fmt.Errorf("value was empty")
	}
	if s.By == "" {
		if strings.HasPrefix(s.Value, "#") {
			s.By = selenium.ByCSSSelector
		} else if strings.HasPrefix(s.Value, ".") {
			s.By = selenium.ByClassName
		} else if strings.Count(s.Value, " ") == 0 {
			s.By = selenium.ByTagName
		} else {
			return fmt.Errorf("by was empty")
		}
	}
	return nil
}

//NewWebElementSelector creates a new instance of web element selector
func NewWebElementSelector(by, value string) *WebElementSelector {
	return &WebElementSelector{
		By:    by,
		Value: value,
	}
}

//RunRequest represents group of selenium web elements calls
type RunRequest struct {
	SessionID      string
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
	Actions        []*Action
}

//RunResponse represents selenium call response
type RunResponse struct {
	SessionID    string
	Data         map[string]interface{}
	LookupErrors []string
}

//MethodCall represents selenium call.
type MethodCall struct {
	Wait       *endly.Repeater
	Method     string
	Parameters []interface{}
}

//Action represents various calls on web element
type Action struct {
	Selector *WebElementSelector
	Calls    []*MethodCall
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
	return nil
}

//NewMethodCall creates a new method call
func NewMethodCall(method string, repeatable *endly.Repeater, parameters ...interface{}) *MethodCall {
	return &MethodCall{
		Wait:       repeatable,
		Method:     method,
		Parameters: parameters,
	}
}

//OpenSessionRequest represents open session request
type OpenSessionRequest struct {
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
}

//Validate validate open session request
func (r *OpenSessionRequest) Validate() error {
	if r.RemoteSelenium == nil {
		return errors.New("Remote (remote selenium endpoint) was empty")
	}
	if r.RemoteSelenium.URL == "" {
		return errors.New("Remote.URL (selenium resource URL) was empty")
	}
	if r.Browser == "" {
		return errors.New("Browser was empty")
	}
	return nil
}
