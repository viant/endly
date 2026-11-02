package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

// Neither shape owns its bounding-box center. The transformed SVG also needs
// scrolling, so local SVG geometry cannot be dispatched as viewport coordinates.
func TestDirectCDPSVGInteriorClickLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Native SVG hit geometry</title>
<style>html{scroll-behavior:smooth}body{height:2400px}svg{display:block;margin-top:1400px;transform:translateX(37px) rotate(13deg);transform-origin:0 0}#overlay{position:fixed;inset:0;z-index:1000}</style>
<button id="html">HTML control</button>
<svg width="480" height="200" viewBox="0 0 240 100">
<path id="annulus" fill="teal" fill-rule="evenodd" d="M90 50a40 40 0 1 0-80 0a40 40 0 1 0 80 0M70 50a20 20 0 1 0-40 0a20 20 0 1 0 40 0"/>
<polygon id="concave" fill="navy" points="110,10 190,10 190,25 125,25 125,75 190,75 190,90 110,90"/>
<rect id="bar" x="205" y="10" width="20" height="80" fill="orange"/>
</svg><div id="overlay"></div><output id="status"></output>
<script>window.clicks=[];for(const id of ['annulus','concave','bar','html'])document.getElementById(id).addEventListener('click',event=>{window.clicks.push({id,trusted:event.isTrusted});document.getElementById('status').textContent=id+':'+event.isTrusted;});</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	for _, id := range []string{"annulus", "concave"} {
		element, err := driver.FindElement(selenium.ByCSSSelector, "#"+id)
		require.NoError(t, err)
		require.Error(t, element.Click(), "overlay must reject %s", id)
	}
	value, err := driver.ExecuteScript(`return window.clicks;`, nil)
	require.NoError(t, err)
	require.Empty(t, value)
	_, err = driver.ExecuteScript(`document.getElementById('overlay').remove();window.scrollTo({top:0,behavior:'instant'});return true;`, nil)
	require.NoError(t, err)
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	putSession(ctx, "svg", &Session{SessionID: "svg", Browser: ChromeBrowser, Backend: "cdp", Attached: true, driver: driver})
	service := New().(*service)
	for _, id := range []string{"annulus", "concave", "bar", "html"} {
		t.Run(id, func(t *testing.T) {
			autoStart := false
			request := &RunRequest{SessionID: "svg", AutoStart: &autoStart, Commands: []interface{}{
				`page.locator("#` + id + `").click()`,
				`expect("#status").toHaveText("` + id + `:true")`,
			}}
			require.NoError(t, request.Init())
			response, err := service.run(ctx, request)
			require.NoError(t, err)
			require.Len(t, response.Assertions, 1)
			require.True(t, response.Assertions[0].Passed)
		})
	}
	value, err = driver.ExecuteScript(`return window.clicks;`, nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{
		map[string]interface{}{"id": "annulus", "trusted": true},
		map[string]interface{}{"id": "concave", "trusted": true},
		map[string]interface{}{"id": "bar", "trusted": true},
		map[string]interface{}{"id": "html", "trusted": true},
	}, value)
}
