package webdriver

import (
	"fmt"
	"strings"

	"github.com/tebeka/selenium"
)

func (s *service) Check(element selenium.WebElement) error   { return setChecked(element, true) }
func (s *service) Uncheck(element selenium.WebElement) error { return setChecked(element, false) }

// File controls are commonly hidden behind a styled label. WebDriver file
// selection does not require a visible click target, unlike ordinary typing.
func (s *service) SetInputFiles(element selenium.WebElement, path string) error {
	tag, err := element.TagName()
	if err != nil {
		return err
	}
	kind, err := element.GetAttribute("type")
	if err != nil {
		return err
	}
	if !strings.EqualFold(tag, "input") || !strings.EqualFold(kind, "file") {
		return fmt.Errorf("setInputFiles requires a file input")
	}
	enabled, err := element.IsEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("file input is disabled")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("setInputFiles requires a file path")
	}
	return element.SendKeys(path)
}

func setChecked(element selenium.WebElement, want bool) error {
	selected, err := element.IsSelected()
	if err != nil {
		return err
	}
	if selected == want {
		return nil
	}
	enabled, err := element.IsEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("checkbox is disabled")
	}
	displayed, err := element.IsDisplayed()
	if err != nil {
		return err
	}
	target := element
	if !displayed {
		labels, err := element.FindElements(selenium.ByXPATH, "ancestor::label[1]")
		if err != nil {
			return err
		}
		if len(labels) == 0 {
			id, err := element.GetAttribute("id")
			if err != nil {
				return err
			}
			if id != "" {
				labels, err = element.FindElements(selenium.ByXPATH, "//label[@for="+xpathLiteral(id)+"]")
				if err != nil {
					return err
				}
			}
		}
		var visible []selenium.WebElement
		for _, label := range labels {
			shown, err := label.IsDisplayed()
			if err != nil {
				return err
			}
			if shown {
				visible = append(visible, label)
			}
		}
		if len(visible) == 0 {
			// Dialog transitions can temporarily hide both the native input
			// and its label. Let the bounded element wait resolve them again.
			return &selenium.Error{LegacyCode: elementNotVisibleException, Err: "element not visible", Message: "checkbox label is not displayed"}
		}
		if len(visible) != 1 {
			return fmt.Errorf("hidden checkbox requires one visible associated label")
		}
		target = visible[0]
	}
	if err = target.Click(); err != nil {
		return err
	}
	selected, err = element.IsSelected()
	if err != nil {
		return err
	}
	if selected != want {
		return &checkboxStatePendingError{}
	}
	return nil
}

// A successful native click may precede a controlled input's render. The
// bounded element retry re-resolves the input and checks its state before
// deciding whether another click is needed.
type checkboxStatePendingError struct{}

func (*checkboxStatePendingError) Error() string {
	return "checkbox state did not change after click"
}

type optionUnavailableError struct{ value string }

func (e *optionUnavailableError) Error() string {
	return fmt.Sprintf("option %q was not found", e.value)
}

// SelectOption selects an option by value first and visible text second.
func (s *service) SelectOption(element selenium.WebElement, value string) error {
	selector := ".//option[@value=" + xpathLiteral(value) + " or normalize-space(.)=" + xpathLiteral(value) + "]"
	options, err := element.FindElements(selenium.ByXPATH, selector)
	if err != nil {
		return err
	}
	if len(options) == 0 {
		return &optionUnavailableError{value: value}
	}
	return options[0].Click()
}

func (s *service) SwitchFrameBySelector(driver selenium.WebDriver, selector string) error {
	by, value := WebSelector(selector).ByAndValue()
	element, err := driver.FindElement(by, value)
	if err != nil {
		return fmt.Errorf("find frame %q: %w", selector, err)
	}
	return driver.SwitchFrame(element)
}

func (s *service) MainFrame(driver selenium.WebDriver) error {
	return driver.SwitchFrame(nil)
}

func (s *service) ElementCount(driver selenium.WebDriver, by, value string) (int, error) {
	elements, err := driver.FindElements(by, value)
	if err != nil {
		return 0, err
	}
	return len(elements), nil
}
