package webdriver

import (
	"fmt"

	"github.com/tebeka/selenium"
)

func (s *service) Check(element selenium.WebElement) error   { return setChecked(element, true) }
func (s *service) Uncheck(element selenium.WebElement) error { return setChecked(element, false) }

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
		return fmt.Errorf("checkbox state did not change after click")
	}
	return nil
}

// SelectOption selects an option by value first and visible text second.
func (s *service) SelectOption(element selenium.WebElement, value string) error {
	selector := ".//option[@value=" + xpathLiteral(value) + " or normalize-space(.)=" + xpathLiteral(value) + "]"
	options, err := element.FindElements(selenium.ByXPATH, selector)
	if err != nil {
		return err
	}
	if len(options) == 0 {
		return fmt.Errorf("option %q was not found", value)
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
