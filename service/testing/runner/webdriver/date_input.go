package webdriver

import (
	"fmt"
	"github.com/tebeka/selenium"
	"strings"
	"time"
)

// fillDate uses Chromium's native segmented date editor. Input.insertText does
// not edit this control; assigning .value would bypass real input behavior.
func fillDate(driver *directCDPDriver, element selenium.WebElement, value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("date fill requires YYYY-MM-DD: %w", err)
	}
	raw, err := driver.ExecuteScript(`const e=arguments[0];if(e.disabled||e.readOnly)throw new Error('date input is not editable');e.focus();return new Intl.DateTimeFormat(e.lang||navigator.language,{year:'numeric',month:'2-digit',day:'2-digit'}).formatToParts(new Date(2026,10,22)).filter(p=>['year','month','day'].includes(p.type)).map(p=>p.type).join(',');`, []interface{}{element})
	if err != nil {
		return err
	}
	values := map[string]string{"year": value[:4], "month": value[5:7], "day": value[8:]}
	key := func(name string, text string) error {
		params := map[string]interface{}{"type": "keyDown", "key": name}
		if text != "" {
			params["text"] = text
			params["unmodifiedText"] = text
		}
		if _, e := driver.CDPCommand("Input.dispatchKeyEvent", params); e != nil {
			return e
		}
		_, e := driver.CDPCommand("Input.dispatchKeyEvent", map[string]interface{}{"type": "keyUp", "key": name})
		return e
	}
	order := strings.Split(fmt.Sprint(raw), ",")
	if len(order) != 3 {
		return fmt.Errorf("unsupported date field order: %v", raw)
	}
	for index, field := range order {
		digits, ok := values[field]
		if !ok {
			return fmt.Errorf("unsupported date field: %s", field)
		}
		// Re-anchor each segment: Chromium can auto-advance after a complete value.
		for n := 0; n < 4; n++ {
			if err = key("ArrowLeft", ""); err != nil {
				return err
			}
		}
		for n := 0; n < index; n++ {
			if err = key("ArrowRight", ""); err != nil {
				return err
			}
		}
		for _, digit := range digits {
			if err = key(string(digit), string(digit)); err != nil {
				return err
			}
		}
	}
	if err = key("Tab", ""); err != nil {
		return err
	}
	actual, err := element.GetAttribute("value")
	if err != nil {
		return err
	}
	if actual != value {
		return fmt.Errorf("native date fill did not settle: got %q, want %q", actual, value)
	}
	return nil
}
