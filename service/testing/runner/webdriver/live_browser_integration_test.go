package webdriver

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

// TestLiveBrowserFlow is opt-in so the default unit suite remains hermetic.
// Set ENDLY_WEBDRIVER_REMOTE to an already-running ChromeDriver endpoint.
// ENDLY_CHROME_DEBUGGER_ADDRESS may identify an existing debug-enabled Chrome.
func TestLiveBrowserFlow(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_REMOTE")
	debuggerAddress := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	directCDP := remote == "" && debuggerAddress != ""
	richEditorBundle := os.Getenv("ENDLY_RICH_EDITOR_BUNDLE")
	if remote == "" && debuggerAddress == "" {
		t.Skip("ENDLY_WEBDRIVER_REMOTE or ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/editor.js" && richEditorBundle != "" {
			http.ServeFile(writer, request, richEditorBundle)
			return
		}
		if request.URL.Path == "/never" {
			writer.Header().Set("Content-Type", "text/html")
			_, _ = writer.Write([]byte(`<!doctype html><html><body><div>Streaming forever</div>`))
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			<-request.Context().Done()
			return
		}
		if request.URL.Path == "/feed" {
			writer.Header().Set("Content-Type", "text/html")
			_, _ = writer.Write([]byte(`<!doctype html><html><body><div style="height:3000px">Feed</div><script>
window.addEventListener('scroll',()=>{const item=document.createElement('div');item.style.height='800px';item.textContent='more';document.body.appendChild(item);});
</script></body></html>`))
			return
		}
		if request.URL.Path == "/api/orders" {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"id":42}`))
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(writer, `<!doctype html><html><body>
<select id="late-options" aria-label="Late options"><option value="">Loading</option></select><script>setTimeout(()=>document.querySelector("#late-options").add(new Option("Ready","ready")),300)</script>
<label onclick="event.preventDefault();setTimeout(()=>document.querySelector('#controlled-check').checked=true,150)">Deferred option<input id="controlled-check" type="checkbox" style="display:none"></label>
<label for="email">Email</label><input id="email" data-testid="email">
<div id="editor" contenteditable="true">Original query</div>
<div id="rich-editor"></div><script src="/editor.js"></script>
<button data-testid="save-editor" onclick="document.querySelector('#saved-editor').textContent=window.endlyRichEditorValue">Save editor</button><div id="saved-editor"></div>
<button data-testid="submit" onclick="fetch('/api/orders',{method:'POST'}).then(()=>document.querySelector('#status').textContent='Signed in')">Submit</button>
<button id="show-toggle" onclick="document.querySelector('#toggle-label').style.display='none';setTimeout(()=>document.querySelector('#toggle-label').style.display='block',300)">Animate checkbox</button><label id="toggle-label">Enabled option<input id="hidden-toggle" type="checkbox" style="display:none"></label><input id="nickname" aria-label="Nickname"><div id="status">Pending</div>
<button id="move-panel" onclick="document.querySelector('#moving-panel').animate([{transform:'translateX(0)'},{transform:'translateX(10px)'}],{duration:1000})">Move dialog</button>
<div id="moving-panel"><label>Moving option<input id="moving-option" type="checkbox" style="display:none" onchange="document.querySelector('#motion-status').textContent=document.querySelector('#moving-panel').getAnimations().some(a=>a.playState==='running')?'moving':'settled'"></label></div><div id="motion-status">Pending</div>
</body></html>`)
	}))
	defer page.Close()

	manager := endly.New()
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	request := &OpenSessionRequest{
		Browser:          ChromeBrowser,
		Remote:           remote,
		DebuggerAddress:  debuggerAddress,
		DirectCDP:        directCDP,
		PageLoadStrategy: "eager",
	}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	if !directCDP {
		putSession(ctx, request.SessionID, &Session{SessionID: request.SessionID, Browser: ChromeBrowser})
	}
	service := New().(*service)
	session, err := service.openSession(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	autoStart := false
	runRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands: []interface{}{
			`page.goto("` + page.URL + `")`,
			`page.getByLabel("Late options").selectOption("ready")`,
			`expect("#late-options").toHaveValue("ready")`,
			`page.locator("#show-toggle").click()`,
			`page.getByLabel("Enabled option").check()`,
			`page.getByLabel("Enabled option").check()`,
			`expect("#hidden-toggle").toBeChecked()`,
			`page.getByLabel("Enabled option").uncheck()`,
			`expect("#hidden-toggle").toBeUnchecked()`,
			`page.getByLabel("Deferred option").check()`,
			`page.getByLabel("Deferred option").check()`,
			`expect("#controlled-check").toBeChecked()`,
			`page.locator("#move-panel").click()`,
			`page.getByLabel("Moving option").check()`,
			`expect("#moving-option").toBeChecked()`,
			`expect("#motion-status").toHaveText("settled", 2000)`,
			`page.getByLabel("Email").fill("qa@example.test")`,
			`page.locator("#editor").fill("SELECT region FROM sales")`,
			`expect("#editor").toHaveText("SELECT region FROM sales", 5000)`,
			`page.locator("#editor").fill("SELECT total FROM sales")`,
			`expect("#editor").toHaveText("SELECT total FROM sales", 5000)`,
			`page.getByLabel("Nickname").fill("Endly")`,
			`expect("#nickname").toHaveValue("Endly", 5000)`,
			`nicknameValue = page.getByLabel("Nickname").inputValue()`,
			`page.getByTestId("submit").click()`,
			`createResponse = page.waitForResponse("/api/orders", 201, 5000)`,
			`expect("#status").toHaveText("Signed in", 5000)`,
			`screenshot = Screenshot()`,
		},
		Navigation: &NavigationOptions{TimeoutMs: 10_000},
	}
	if richEditorBundle != "" {
		runRequest.Commands = append(runRequest.Commands,
			`page.locator(".cm-content").fill("SELECT region FROM sales")`,
			`page.getByTestId("save-editor").click()`,
			`expect("#saved-editor").toHaveText("SELECT region FROM sales", 5000)`,
			`page.locator(".cm-content").fill("SELECT total FROM sales")`,
			`page.getByTestId("save-editor").click()`,
			`expect("#saved-editor").toHaveText("SELECT total FROM sales", 5000)`,
		)
	}
	if err := runRequest.Init(); err != nil {
		t.Fatal(err)
	}
	response, err := service.run(ctx, runRequest)
	if err != nil {
		t.Fatal(err)
	}
	expectedAssertions := 10
	if richEditorBundle != "" {
		expectedAssertions += 2
	}
	if len(response.Assertions) != expectedAssertions {
		t.Fatalf("assertions=%#v", response.Assertions)
	}
	for _, assertion := range response.Assertions {
		if !assertion.Passed {
			t.Fatalf("assertion failed: %#v", assertion)
		}
	}
	shot, ok := response.Data["screenshot"].([]byte)
	if !ok {
		t.Fatalf("screenshot lost binary type: %T", response.Data["screenshot"])
	}
	if picture, err := png.Decode(bytes.NewReader(shot)); err != nil || picture.Bounds().Dx() == 0 {
		t.Fatalf("invalid browser PNG: %v", err)
	}
	if response.Data["createResponse"] == nil {
		t.Fatalf("browser response was not captured: %#v", response.Data)
	}
	if response.Data["nicknameValue"] != "Endly" {
		t.Fatalf("inputValue did not return the live DOM value: %v", response.Data["nicknameValue"])
	}
	// A control removed by a render must report a retryable element error,
	// rather than an opaque JavaScript exception that aborts the workflow.
	element, err := session.driver.FindElement(selenium.ByCSSSelector, "#nickname")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = session.driver.ExecuteScript(`arguments[0].remove();`, []interface{}{element}); err != nil {
		t.Fatal(err)
	}
	_, err = element.GetProperty("value")
	var stale *selenium.Error
	if !errors.As(err, &stale) || stale.Err != "stale element reference" || !IsRetryableElementError(err) {
		t.Fatalf("removed control did not return a retryable stale-element error: %v", err)
	}

	// A child lookup must also classify a removed parent as stale.
	selectElement, err := session.driver.FindElement(selenium.ByCSSSelector, "#late-options")
	if err != nil {
		t.Fatal(err)
	}
	option, err := selectElement.FindElement(selenium.ByXPATH, ".//option[@value='ready']")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = session.driver.ExecuteScript(`arguments[0].remove();`, []interface{}{selectElement}); err != nil {
		t.Fatal(err)
	}
	if err = option.Click(); !IsRetryableElementError(err) {
		t.Fatalf("removed select parent did not return a retryable error: %v", err)
	}

	tabRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands: []interface{}{
			`page.newTab("` + page.URL + `/tab")`,
			`tabs = page.tabs()`,
			`expect(page).toHaveURL("` + page.URL + `/tab", 2000)`,
			`page.closeTab()`,
		},
	}
	if err := tabRequest.Init(); err != nil {
		t.Fatal(err)
	}
	tabResponse, err := service.run(ctx, tabRequest)
	if err != nil {
		t.Fatal(err)
	}
	if tabResponse.Data["tabs"] == nil || len(tabResponse.Assertions) != 1 || !tabResponse.Assertions[0].Passed {
		t.Fatalf("tab response=%#v", tabResponse)
	}

	neverRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands:  []interface{}{`page.goto("` + page.URL + `/never")`},
		Navigation: &NavigationOptions{
			TimeoutMs: 300,
		},
	}
	if err := neverRequest.Init(); err != nil {
		t.Fatal(err)
	}
	neverResponse, err := service.run(ctx, neverRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(neverResponse.Navigations) != 1 || !neverResponse.Navigations[0].TimedOut || !neverResponse.Navigations[0].LoadingStopped {
		t.Fatalf("never-loading report=%#v", neverResponse.Navigations)
	}

	feedRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands:  []interface{}{`page.goto("` + page.URL + `/feed")`},
		Navigation: &NavigationOptions{
			TimeoutMs:         2_000,
			AutoScrollMs:      2_000,
			ScrollDelayMs:     25,
			StableWindowMs:    500,
			MaxScrollSteps:    3,
			MaxScrollGrowthPx: 100_000,
		},
	}
	if err := feedRequest.Init(); err != nil {
		t.Fatal(err)
	}
	feedResponse, err := service.run(ctx, feedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(feedResponse.Navigations) != 1 || feedResponse.Navigations[0].Steps > 3 || feedResponse.Navigations[0].StopReason != "max-steps" {
		t.Fatalf("infinite-feed report=%#v", feedResponse.Navigations)
	}
}
