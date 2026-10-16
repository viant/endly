package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

// Use only an explicitly owned Chrome debug endpoint. The disposable local
// page models controlled state; edits must come from trusted native input.
func TestDirectCDPControlledFillLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Endly controlled fill fixture</title><input id="phone" type="tel" value="(555) 123-4567"><input id="date" type="date" value="2026-01-01"><textarea id="document">old document</textarea><div id="editor" contenteditable="plaintext-only" style="white-space:pre-wrap">old editor</div><script>
 const date=document.querySelector('#date');window.dateEdits=[];date.addEventListener('input',()=>window.dateEdits.push(date.value));
 const phone=document.querySelector('#phone');let controlled=phone.value;window.inputs=[];
 phone.addEventListener('input',e=>{window.inputs.push({trusted:e.isTrusted,type:e.inputType});controlled=phone.value;requestAnimationFrame(()=>{phone.value=controlled})});
 </script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	phone, err := driver.FindElement(selenium.ByCSSSelector, "#phone")
	require.NoError(t, err)
	require.NoError(t, fillElement(driver, phone, ""))
	value, err := driver.ExecuteScript(`return new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve({value:document.querySelector('#phone').value,inputs:window.inputs}))));`, nil)
	require.NoError(t, err)
	state := value.(map[string]interface{})
	require.Empty(t, state["value"])
	inputs := state["inputs"].([]interface{})
	require.Len(t, inputs, 1)
	event := inputs[0].(map[string]interface{})
	require.Equal(t, true, event["trusted"])
	require.Equal(t, "deleteContentBackward", event["type"])
	require.NoError(t, fillElement(driver, phone, "1234"))
	require.NoError(t, phone.SendKeys(selenium.LeftArrowKey))
	require.NoError(t, phone.SendKeys(selenium.DeleteKey))
	value, err = driver.ExecuteScript(`return document.querySelector('#phone').value;`, nil)
	require.NoError(t, err)
	require.Equal(t, "123", value)
	date, err := driver.FindElement(selenium.ByCSSSelector, "#date")
	require.NoError(t, err)
	require.NoError(t, fillElement(driver, date, "2026-10-13"))
	value, err = driver.ExecuteScript(`return document.querySelector('#date').value;`, nil)
	require.NoError(t, err)
	require.Equal(t, "2026-10-13", value)
	value, err = driver.ExecuteScript(`return window.dateEdits;`, nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{"2026-10-13"}, value)
	require.Error(t, fillElement(driver, date, "not-a-date"))
	value, err = driver.ExecuteScript(`return document.querySelector('#date').value;`, nil)
	require.NoError(t, err)
	require.Equal(t, "2026-10-13", value)
	require.NoError(t, fillElement(driver, date, ""))
	value, err = driver.ExecuteScript(`return document.querySelector('#date').value;`, nil)
	require.NoError(t, err)
	require.Equal(t, "", value)
	document := "profile:\n  id: customer-intent\n  description: Preserve indentation."
	for _, selector := range []string{"#document", "#editor"} {
		field, err := driver.FindElement(selenium.ByCSSSelector, selector)
		require.NoError(t, err)
		require.NoError(t, fillElement(driver, field, document))
		value, err := driver.ExecuteScript(`const e=arguments[0];return e.value===undefined?e.innerText:e.value;`, []interface{}{field})
		require.NoError(t, err)
		require.Equal(t, document, value)
	}
	t.Log("trusted native deletion cleared controlled Phone; ArrowLeft/Delete edited text; textarea and contenteditable preserved multiline indentation")
}
