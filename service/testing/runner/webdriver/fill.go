package webdriver

import (
	"fmt"
	"github.com/tebeka/selenium"
	"strings"
)

// Select before typing so controlled inputs observe one replacement input event.
// Native Clear can blur the field and restore its old controlled value before
// SendKeys runs, causing the new value to be appended.
func fillElement(driver selenium.WebDriver, element selenium.WebElement, value string, insertText ...func(string) error) error {
	selection, err := driver.ExecuteScript(`const e=arguments[0];
if(e.disabled||e.readOnly||(!e.isContentEditable&&!['INPUT','TEXTAREA'].includes(e.tagName)))throw new Error('fill requires an editable text control');
if(e.tagName==='INPUT'&&['checkbox','radio','file','button','submit','reset','range','color','hidden'].includes(e.type))throw new Error('fill requires a text input');
e.focus();
if(e.isContentEditable)return /Mac/.test(navigator.platform)?'contenteditable-meta':'contenteditable-control';
if(typeof e.select==='function'){e.select();if(e.selectionStart===0&&e.selectionEnd===e.value.length)return 'selected';}
return /Mac/.test(navigator.platform)?'meta':'control';`, []interface{}{element, value})
	if err != nil {
		return err
	}
	if selection != "selected" {
		modifier := selenium.ControlKey
		if selection == "meta" || selection == "contenteditable-meta" {
			modifier = selenium.MetaKey
		}
		if direct, ok := driver.(*directCDPDriver); ok {
			mask := 2
			if modifier == selenium.MetaKey {
				mask = 4
			}
			for _, kind := range []string{"keyDown", "keyUp"} {
				params := map[string]interface{}{"type": kind, "key": "a", "code": "KeyA", "windowsVirtualKeyCode": 65, "modifiers": mask}
				if kind == "keyDown" {
					params["commands"] = []string{"selectAll"}
				}
				if _, err = direct.CDPCommand("Input.dispatchKeyEvent", params); err != nil {
					return err
				}
			}
		} else if err = element.SendKeys(modifier + "a" + selenium.NullKey); err != nil {
			return err
		}
	}
	// Replace editable documents as one native input, rather than typing Enter
	// keys that activate editor indentation and other keyboard commands.
	if strings.HasPrefix(fmt.Sprint(selection), "contenteditable") && value != "" {
		if len(insertText) > 0 {
			if err = insertText[0](value); err == nil {
				return nil
			}
		}
		if _, direct := driver.(*directCDPDriver); direct {
			return element.SendKeys(value)
		}
		_, err = driver.ExecuteScript(`if(!document.execCommand('insertText',false,arguments[1]))throw new Error('native editable text insertion failed');`, []interface{}{element, value})
		return err
	}
	if value == "" {
		value = selenium.BackspaceKey
	}
	return element.SendKeys(value)
}
