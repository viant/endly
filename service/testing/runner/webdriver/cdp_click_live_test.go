package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

func TestDirectCDPClickRejectsOverlayAndScrollsInstantlyLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Endly native hit target fixture</title><style>html{scroll-behavior:smooth}body{height:2200px}#save{margin-top:1500px}#overlay{position:fixed;inset:0;z-index:1000}</style><button id="save">Save</button><div id="overlay"></div><script>window.clicks=[];document.querySelector('#save').addEventListener('click',e=>window.clicks.push({trusted:e.isTrusted}));</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	button, err := driver.FindElement(selenium.ByCSSSelector, "#save")
	require.NoError(t, err)
	err = button.Click()
	require.Error(t, err)
	require.True(t, IsRetryableElementError(err))
	result, err := driver.ExecuteScript(`return window.clicks.length;`, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, result)
	// Fixture setup removes the obstruction; the actual action remains trusted CDP input.
	_, err = driver.ExecuteScript(`document.querySelector('#overlay').remove();window.scrollTo({top:0,behavior:'instant'});return true;`, nil)
	require.NoError(t, err)
	require.NoError(t, button.Click())
	result, err = driver.ExecuteScript(`return {count:window.clicks.length,trusted:window.clicks[0]?.trusted,scrolled:window.scrollY>0};`, nil)
	require.NoError(t, err)
	state := result.(map[string]interface{})
	require.EqualValues(t, 1, state["count"])
	require.Equal(t, true, state["trusted"])
	require.Equal(t, true, state["scrolled"])
}
