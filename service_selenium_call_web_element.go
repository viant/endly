package endly

import (
	"time"
	"fmt"
	"strings"
	"github.com/tebeka/selenium"
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

func (r *SeleniumWebElementCallRequest) Data() (int, time.Duration, string) {
	var repeat = 1
	var sleepInMs = time.Millisecond * 0
	var exitCriteria = ""

	var wait = r.Call.Wait
	if wait != nil {
		if wait.Repeat > 0 {
			repeat = wait.Repeat
		}
		sleepInMs = time.Duration(wait.SleepInMs) * time.Millisecond
		exitCriteria = wait.ExitCriteria
	}
	return repeat, sleepInMs, exitCriteria
}

//SeleniumWait represents selenium wait data
type SeleniumWait struct {
	Repeat       int
	SleepInMs    int
	ExitCriteria string
}

//SeleniumWebElementCallResponse represents seleniun web element response
type SeleniumWebElementCallResponse struct {
	Result []interface{}
}

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
func NewWebElementSelector(by, value string) *WebElementSelector{
	return &WebElementSelector{
		By:by,
		Value:value,
	}
}