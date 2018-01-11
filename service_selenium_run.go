package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//SeleniumRunRequest represents group of selenium web elements calls
type SeleniumRunRequest struct {
	SessionID      string
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
	Actions        []*SeleniumAction
}

//SeleniumRunResponse represents selenium call response
type SeleniumRunResponse struct {
	SessionID    string
	Data         map[string]interface{}
	LookupErrors []string
}

//SeleniumMethodCall represents selenium call.
type SeleniumMethodCall struct {
	Wait       *Repeatable
	Method     string
	Parameters []interface{}
}

//SeleniumAction represents various calls on web element
type SeleniumAction struct {
	Selector *WebElementSelector
	Calls    []*SeleniumMethodCall
}

//Validate validates run request.
func (r *SeleniumRunRequest) Validate() error {
	if r.SessionID == "" {
		if r.RemoteSelenium == nil {
			return fmt.Errorf("both SessionID and RemoteSelenium were empty")
		}
		if r.Browser == "" {
			return fmt.Errorf("both SessionID and Browser were empty")
		}
	}
	return nil
}

//NewSeleniumMethodCall creates a new method call
func NewSeleniumMethodCall(method string, repeatable *Repeatable, parameters ...interface{}) *SeleniumMethodCall {
	return &SeleniumMethodCall{
		Wait:       repeatable,
		Method:     method,
		Parameters: parameters,
	}
}
