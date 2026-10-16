package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

func TestDirectCDPARIACheckedLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Endly ARIA checked fixture</title>
 <button id="switch" role="switch" aria-checked="false">Switch</button>
 <button id="checkbox" role="checkbox" aria-checked="false">Checkbox</button>
 <button id="radio" role="radio" aria-checked="false">Radio</button>
 <input id="native" type="checkbox">
 <script>window.clicks=[];document.querySelectorAll('button').forEach(e=>e.addEventListener('click',event=>{window.clicks.push({id:e.id,trusted:event.isTrusted});e.setAttribute('aria-checked',String(e.getAttribute('aria-checked')!=='true'));}));</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	for _, id := range []string{"switch", "checkbox", "radio", "native"} {
		element, err := driver.FindElement(selenium.ByCSSSelector, "#"+id)
		require.NoError(t, err)
		checked, err := element.IsSelected()
		require.NoError(t, err)
		require.False(t, checked)
		require.NoError(t, setChecked(element, true))
		require.NoError(t, setChecked(element, true), "checking twice must be idempotent")
		checked, err = element.IsSelected()
		require.NoError(t, err)
		require.True(t, checked)
		require.NoError(t, setChecked(element, false))
		require.NoError(t, setChecked(element, false), "unchecking twice must be idempotent")
		checked, err = element.IsSelected()
		require.NoError(t, err)
		require.False(t, checked)
	}
	value, err := driver.ExecuteScript(`return {count:window.clicks.length,trusted:window.clicks.every(e=>e.trusted)};`, nil)
	require.NoError(t, err)
	result := value.(map[string]interface{})
	require.EqualValues(t, 6, result["count"])
	require.Equal(t, true, result["trusted"])
}
