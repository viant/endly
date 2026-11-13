package webdriver

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

const directCDPNestedMenuFixture = `<!doctype html><title>Nested menu hit target</title><style>
body{margin:0}#rail{position:fixed;right:0;top:0;width:300px;height:700px;overflow:auto}
#menu{margin-top:300px;height:300px;overflow:auto}#content{height:1200px;padding-top:240px}
#target{display:block;margin:0 0 0 40px;width:13px;height:13px}#tail{height:1400px}
#toolbar{position:fixed;right:0;top:0;width:300px;height:400px;z-index:10;background:#ddd}
</style><div id="rail"><div id="menu"><div id="content"><input id="target" type="checkbox" aria-label="Nested choice"></div></div><div id="tail"></div></div><div id="toolbar">Toolbar</div>
<script>window.clicks=[];target.addEventListener('click',e=>clicks.push(e.isTrusted));</script>`

// Centering every ancestor can hide an already actionable menu control behind
// fixed page chrome. The checkbox starts visible inside two independent scrollers.
func TestDirectCDPClickPreservesVisibleNestedMenuLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(directCDPNestedMenuFixture))
	require.NoError(t, err)
	defer driver.Close()
	_, err = driver.CDPCommand("Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	require.NoError(t, err)
	target, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	require.NoError(t, err)
	value, err := driver.ExecuteScript(`const r=target.getBoundingClientRect();return document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===target;`, nil)
	require.NoError(t, err)
	require.Equal(t, true, value, "fixture must start with the checkbox owning its center")
	require.NoError(t, target.Click())
	value, err = driver.ExecuteScript(`return {checked:target.checked,clicks:window.clicks,rail:rail.scrollTop,menu:menu.scrollTop};`, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"checked": true, "clicks": []interface{}{true}, "rail": float64(0), "menu": float64(0)}, value)
}

// The public run path must preserve the same native click actionability as
// direct element.Click; a service pre-scroll must not hide the visible input.
func TestDirectCDPRunClickPreservesVisibleNestedMenuLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(directCDPNestedMenuFixture))
	require.NoError(t, err)
	defer driver.Close()
	_, err = driver.CDPCommand("Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	require.NoError(t, err)
	value, err := driver.ExecuteScript(`const r=target.getBoundingClientRect();return document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===target;`, nil)
	require.NoError(t, err)
	require.Equal(t, true, value, "fixture must start with the checkbox owning its center")
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	putSession(ctx, "nested-menu", &Session{SessionID: "nested-menu", Browser: ChromeBrowser, Backend: "cdp", Attached: true, driver: driver})
	autoStart := false
	request := &RunRequest{SessionID: "nested-menu", AutoStart: &autoStart, ActionTimeoutMs: 500, Commands: []interface{}{
		`expect("#target").toBeVisible()`,
		`expect("#target").toBeEnabled()`,
		`page.getByLabel("Nested choice").click()`,
		`expect("#target").toBeChecked()`,
	}}
	require.NoError(t, request.Init())
	response, err := New().(*service).run(ctx, request)
	require.NoError(t, err)
	require.Len(t, response.Assertions, 3)
	for _, assertion := range response.Assertions {
		require.True(t, assertion.Passed)
	}
	value, err = driver.ExecuteScript(`return {checked:target.checked,clicks:window.clicks,rail:rail.scrollTop,menu:menu.scrollTop};`, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"checked": true, "clicks": []interface{}{true}, "rail": float64(0), "menu": float64(0)}, value)
}

// A target above the viewport needs the original center-scroll fallback:
// nearest scrolling can leave its center covered by a fixed header.
func TestDirectCDPClickScrollsAboveViewportPastFixedHeaderLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Offscreen target and fixed header</title><style>
body{margin:0;height:3000px}#target{position:absolute;top:500px;left:40px;width:100px;height:30px}
header{position:fixed;top:0;left:0;right:0;height:100px;z-index:10;background:#ddd}
</style><button id="target">Target</button><header>Toolbar</header>
<script>window.clicks=[];target.addEventListener('click',e=>clicks.push(e.isTrusted));</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	_, err = driver.CDPCommand("Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	require.NoError(t, err)
	value, err := driver.ExecuteScript(`window.scrollTo({top:1000,behavior:'instant'});return target.getBoundingClientRect().bottom<0;`, nil)
	require.NoError(t, err)
	require.Equal(t, true, value, "fixture must start above the viewport")
	target, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	require.NoError(t, err)
	require.NoError(t, target.Click())
	value, err = driver.ExecuteScript(`const r=target.getBoundingClientRect();return {clicks:window.clicks,uncovered:r.top>=100&&document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===target};`, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"clicks": []interface{}{true}, "uncovered": true}, value)
}

func TestDirectCDPClickPointStillScrollsAndRejectsUnsafeTargetsLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Nested clipping and motion</title><style>
body{margin:0;height:2600px}#rail{margin-top:1300px;height:300px;overflow:auto}
#menu{margin-top:700px;height:200px;overflow:auto}#content{height:1400px;padding-top:600px}
#target{display:block;width:20px;height:20px;margin:0 0 0 20px}#tail{height:1000px}
#overlay{position:fixed;inset:0;z-index:10}
</style><div id="rail"><div id="menu"><div id="content"><input id="target" type="checkbox"></div></div><div id="tail"></div></div>
<script>window.clicks=[];target.addEventListener('click',e=>clicks.push(e.isTrusted));</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	_, err = driver.CDPCommand("Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false})
	require.NoError(t, err)
	element, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	require.NoError(t, err)
	target := element.(*directCDPElement)
	require.NoError(t, target.Click())
	value, err := driver.ExecuteScript(`return {checked:target.checked,clicks:window.clicks,scrolled:scrollY>0&&rail.scrollTop>0&&menu.scrollTop>0};`, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"checked": true, "clicks": []interface{}{true}, "scrolled": true}, value)
	_, err = driver.ExecuteScript(`const overlay=document.createElement('div');overlay.id='overlay';document.body.append(overlay);return true;`, nil)
	require.NoError(t, err)
	_, _, err = target.clickableCenter(150 * time.Millisecond)
	require.ErrorContains(t, err, "obscured or moving")
	_, err = driver.ExecuteScript(`overlay.remove();let x=0;window.move=()=>{target.style.transform='translateX('+(x+=2)+'px)';window.frame=requestAnimationFrame(window.move);};window.move();return true;`, nil)
	require.NoError(t, err)
	_, _, err = target.clickableCenter(150 * time.Millisecond)
	require.ErrorContains(t, err, "obscured or moving")
	_, err = driver.ExecuteScript(`cancelAnimationFrame(window.frame);target.remove();return true;`, nil)
	require.NoError(t, err)
	_, _, err = target.clickableCenter(150 * time.Millisecond)
	require.Error(t, err)
	value, err = driver.ExecuteScript(`return window.clicks;`, nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{true}, value, "unsafe targets must not receive further clicks")
}

func TestDirectCDPHoverWithoutAnimationFramesLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<svg width="200" height="100"><circle id="mark" cx="100" cy="50" r="6"/></svg>
<script>window.requestAnimationFrame=()=>0;mark.onmousemove=e=>window.hovered=e.isTrusted</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	mark, err := driver.FindElement(selenium.ByCSSSelector, "#mark")
	require.NoError(t, err)
	require.NoError(t, mark.MoveTo(0, 0))
	value, err := driver.ExecuteScript("return window.hovered", nil)
	require.NoError(t, err)
	require.Equal(t, true, value)
}
