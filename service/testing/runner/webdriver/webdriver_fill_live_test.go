package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
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
