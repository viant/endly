package webdriver

import (
	"github.com/tebeka/selenium"
)

// Select before typing so controlled inputs observe one replacement input event.
// Native Clear can blur the field and restore its old controlled value before
// SendKeys runs, causing the new value to be appended.
func fillElement(driver selenium.WebDriver, element selenium.WebElement, value string) error {
	selection, err := driver.ExecuteScript(`const e=arguments[0];
if(e.disabled||e.readOnly||(!e.isContentEditable&&!['INPUT','TEXTAREA'].includes(e.tagName)))throw new Error('fill requires an editable text control');
if(e.tagName==='INPUT'&&['checkbox','radio','file','button','submit','reset','range','color','hidden'].includes(e.type))throw new Error('fill requires a text input');
e.focus();
if(e.isContentEditable){const r=document.createRange();r.selectNodeContents(e);const s=window.getSelection();s.removeAllRanges();s.addRange(r);return 'selected';}
if(typeof e.select==='function'){e.select();if(e.selectionStart===0&&e.selectionEnd===e.value.length)return 'selected';}
return /Mac/.test(navigator.platform)?'meta':'control';`, []interface{}{element})
	if err != nil {
		return err
	}
	if selection != "selected" {
		modifier := selenium.ControlKey
		if selection == "meta" {
			modifier = selenium.MetaKey
		}
		if direct, ok := driver.(*directCDPDriver); ok {
			mask := 2
			if modifier == selenium.MetaKey {
				mask = 4
			}
			for _, kind := range []string{"keyDown", "keyUp"} {
				if _, err = direct.CDPCommand("Input.dispatchKeyEvent", map[string]interface{}{"type": kind, "key": "a", "code": "KeyA", "modifiers": mask}); err != nil {
					return err
				}
			}
		} else if err = element.SendKeys(modifier + "a" + selenium.NullKey); err != nil {
			return err
		}
	}
	if value == "" {
		value = selenium.BackspaceKey
	}
	return element.SendKeys(value)
}
