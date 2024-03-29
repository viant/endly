package selenium

import (
	"fmt"
	"github.com/tebeka/selenium"
	"strings"
)

type WebSelector string

var selectors = map[string]bool{
	selenium.ByCSSSelector:     true,
	selenium.ByClassName:       true,
	selenium.ByTagName:         true,
	selenium.ByXPATH:           true,
	selenium.ByID:              true,
	selenium.ByLinkText:        true,
	selenium.ByPartialLinkText: true,
}

// Validate checks is selector is valid.
func (s *WebElementSelector) Init() error {
	if s.Value != "" && s.By == "" {
		s.By, _ = WebSelector(s.Value).ByAndValue()
	}
	return nil
}

// Validate checks is selector is valid.
func (s *WebElementSelector) Validate() error {
	if s.Value == "" {
		return fmt.Errorf("value was empty")
	}

	return nil
}

// NewWebElementSelector creates a new instance of web element selector
func NewWebElementSelector(by, value string) *WebElementSelector {
	return &WebElementSelector{
		By:    by,
		Value: value,
	}
}

func (s WebSelector) ByAndValue() (by, value string) {
	var selector = string(s)
	var byIndex = strings.Index(selector, ":")
	if byIndex != -1 {
		var byCandidate = strings.TrimSpace(string(selector[:byIndex]))
		if selectors[byCandidate] {
			return byCandidate, strings.TrimSpace(string(selector[byIndex+1:]))
		}
	}
	if strings.HasPrefix(selector, "#") {
		return selenium.ByCSSSelector, selector
	} else if strings.HasPrefix(selector, ".") {
		return selenium.ByClassName, selector
	} else if !(strings.Count(selector, " ") > 0 || strings.Count(selector, "[") > 0 || strings.Count(selector, "//") > 0) {
		return selenium.ByTagName, selector
	}
	return selenium.ByXPATH, selector
}
