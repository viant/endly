package endly

import (
	"fmt"
	"github.com/tebeka/selenium"
	"strings"
)

//WebElementSelector represents a web element selector
type WebElementSelector struct {
	By    string //selector type
	Value string //selector value
	Key   string //optional result key, otherwise value is used
}

//SeleniumWebElementCallRequest represents a web element call reqesut
type SeleniumWebElementCallRequest struct {
	SessionID string
	Selector  *WebElementSelector
	Call      *SeleniumMethodCall
}

//SeleniumWebElementCallResponse represents seleniun web element response
type SeleniumWebElementCallResponse struct {
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
