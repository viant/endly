package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

func TestWebDriverFillSurvivesControlledInputReplacement(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_TEST_URL")
	if remote == "" {
		t.Skip("owned ChromeDriver endpoint required")
	}
	driver, err := selenium.NewRemote(selenium.Capabilities{"browserName": "chrome"}, remote)
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Quit() })
	page := `<!doctype html><title>Controlled input fixture</title><input id="field" value="old"><script>
window.edits=[];
function bind(field){field.addEventListener('input',event=>{
 window.edits.push({value:field.value,trusted:event.isTrusted});
 const next=field.cloneNode();next.value=field.value;field.replaceWith(next);bind(next);
});}bind(document.querySelector('#field'));
</script>`
	require.NoError(t, driver.Get("data:text/html,"+url.PathEscape(page)))
	element, err := driver.FindElement(selenium.ByCSSSelector, "#field")
	require.NoError(t, err)
	session := &Session{driver: driver, Remote: remote}
	value := "child.parent_id=parent.id"
	require.NoError(t, fillElement(driver, element, value, func(text string) error {
		_, err := executeSessionCDP(session, "Input.insertText", map[string]any{"text": text})
		return err
	}))
	actual, err := driver.ExecuteScript(`return {value:document.querySelector('#field').value,edits:window.edits};`, nil)
	require.NoError(t, err)
	state := actual.(map[string]interface{})
	require.Equal(t, value, state["value"])
	edits := state["edits"].([]interface{})
	require.Len(t, edits, 1)
	require.Equal(t, true, edits[0].(map[string]interface{})["trusted"])
}

func TestWebDriverFillRejectsInheritedDisabledFieldset(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_TEST_URL")
	if remote == "" {
		t.Skip("owned ChromeDriver endpoint required")
	}
	driver, err := selenium.NewRemote(selenium.Capabilities{"browserName": "chrome", "goog:chromeOptions": map[string]any{"args": []string{"--headless=new"}}}, remote)
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Quit() })
	page := `<!doctype html><input id="other"><fieldset disabled><input id="field"></fieldset><script>document.querySelector('#other').focus();</script>`
	require.NoError(t, driver.Get("data:text/html,"+url.PathEscape(page)))
	element, err := driver.FindElement(selenium.ByCSSSelector, "#field")
	require.NoError(t, err)
	session := &Session{driver: driver, Remote: remote}
	err = fillElement(driver, element, "blocked", func(text string) error {
		_, err := executeSessionCDP(session, "Input.insertText", map[string]any{"text": text})
		return err
	})
	require.Error(t, err, "a disabled ancestor must not be reported as a successful fill")
	actual, err := driver.ExecuteScript(`return [document.querySelector('#field').value,document.querySelector('#other').value];`, nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{"", ""}, actual, "typing must not be redirected to another focused control")
}

func TestWebDriverFillWaitsForEffectiveEditability(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_TEST_URL")
	if remote == "" {
		t.Skip("owned ChromeDriver endpoint required")
	}
	for _, tc := range []struct{ name, markup, enable string }{
		{"fieldset", `<fieldset disabled><input id="field"></fieldset>`, `document.querySelector('fieldset').disabled=false`},
		{"readonly", `<input id="field" readonly>`, `document.querySelector('#field').readOnly=false`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := selenium.NewRemote(selenium.Capabilities{"browserName": "chrome", "goog:chromeOptions": map[string]any{"args": []string{"--headless=new"}}}, remote)
			require.NoError(t, err)
			t.Cleanup(func() { _ = driver.Quit() })
			ctx := endly.New().NewContext(nil)
			defer ctx.Close()
			putSession(ctx, "editable", &Session{SessionID: "editable", Browser: ChromeBrowser, driver: driver, Remote: remote})
			require.NoError(t, driver.Get("data:text/html,"+url.PathEscape(`<!doctype html><input id="other">`+tc.markup)))
			_, err = driver.ExecuteScript(`document.querySelector('#other').focus();setTimeout(()=>{`+tc.enable+`;},250);`, nil)
			require.NoError(t, err)
			autoStart := false
			request := &RunRequest{SessionID: "editable", AutoStart: &autoStart, ActionTimeoutMs: 2000, PollIntervalMs: 10, Commands: []interface{}{`page.locator("#field").fill("ready")`, `expect("#field").toHaveValue("ready")`, `expect("#other").toHaveValue("")`}}
			require.NoError(t, request.Init())
			response, err := New().(*service).run(ctx, request)
			require.NoError(t, err)
			require.Len(t, response.Assertions, 2)
			for _, assertion := range response.Assertions {
				require.True(t, assertion.Passed)
			}
		})
	}
}
