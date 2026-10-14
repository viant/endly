package webdriver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

type fillTestDriver struct {
	selenium.WebDriver
	mode string
	err  error
}

func (d *fillTestDriver) ExecuteScript(_ string, args []interface{}) (interface{}, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.mode == "selected" {
		args[0].(*fillTestElement).selected = true
	}
	return d.mode, nil
}

type fillTestElement struct {
	selenium.WebElement
	value    string
	selected bool
}

func (e *fillTestElement) Clear() error {
	panic("fill must not blur a controlled input through native Clear")
}
func (e *fillTestElement) SendKeys(keys string) error {
	if keys == selenium.ControlKey+"a"+selenium.NullKey || keys == selenium.MetaKey+"a"+selenium.NullKey {
		e.selected = true
		return nil
	}
	if e.selected {
		e.value = ""
		e.selected = false
	}
	if keys != selenium.BackspaceKey {
		e.value += keys
	}
	return nil
}
func TestFillReplacesControlledValueWithoutClear(t *testing.T) {
	for _, mode := range []string{"selected", "control", "meta"} {
		t.Run(mode, func(t *testing.T) {
			element := &fillTestElement{value: "previous title"}
			driver := &fillTestDriver{mode: mode}
			require.NoError(t, fillElement(driver, element, "new title"))
			require.Equal(t, "new title", element.value)
			require.NoError(t, fillElement(driver, element, ""))
			require.Empty(t, element.value)
		})
	}
}
func TestFillStopsWhenSelectionFails(t *testing.T) {
	element := &fillTestElement{value: "previous title"}
	err := errors.New("stale selection")
	require.ErrorIs(t, fillElement(&fillTestDriver{err: err}, element, "new title"), err)
	require.Equal(t, "previous title", element.value)
}
