package webdriver

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

type fillTestDriver struct {
	selenium.WebDriver
	mode string
	err  error
}

func (d *fillTestDriver) ExecuteScript(script string, args []interface{}) (interface{}, error) {
	if d.err != nil {
		return nil, d.err
	}
	if len(args) == 2 && !strings.Contains(script, "fill requires") {
		args[0].(*fillTestElement).value = args[1].(string)
		return true, nil
	}
	if d.mode == "selected" || d.mode == "contenteditable" {
		args[0].(*fillTestElement).selected = true
	}
	return d.mode, nil
}

func TestFillEditableDocumentPreservesMultilineIndentation(t *testing.T) {
	element := &fillTestElement{value: "previous document"}
	document := "schemaVersion: 1\nprofile:\n  id: customer-intent\n  instructions: Read approved data.\n"
	require.NoError(t, fillElement(&fillTestDriver{mode: "contenteditable"}, element, document))
	require.Equal(t, document, element.value)
}

func TestFillEditableDocumentUsesNativeInsertionWhenAvailable(t *testing.T) {
	element := &fillTestElement{value: "previous document"}
	document := "profile:\n  id: customer-intent\n  instructions: Read approved data.\n"
	called := false
	require.NoError(t, fillElement(&fillTestDriver{mode: "contenteditable"}, element, document, func(text string) error {
		called = true
		require.Equal(t, document, text)
		element.value = text
		return nil
	}))
	require.True(t, called)
	require.Equal(t, document, element.value)
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

func TestFillTemporalControlUsesDOMEditEvents(t *testing.T) {
	for _, value := range []string{"2026-10-13", ""} {
		element := &fillTestElement{value: "2026-01-01"}
		require.NoError(t, fillElement(&fillTestDriver{mode: "temporal"}, element, value))
		require.Equal(t, value, element.value)
	}
}
