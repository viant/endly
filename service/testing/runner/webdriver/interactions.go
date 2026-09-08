package webdriver

import (
	"fmt"

	"github.com/tebeka/selenium"
)

func (s *service) Check(element selenium.WebElement) error {
	selected, err := element.IsSelected()
	if err != nil {
		return err
	}
	if selected {
		return nil
	}
	return element.Click()
}

func (s *service) Uncheck(element selenium.WebElement) error {
	selected, err := element.IsSelected()
	if err != nil {
		return err
	}
	if !selected {
		return nil
	}
	return element.Click()
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
