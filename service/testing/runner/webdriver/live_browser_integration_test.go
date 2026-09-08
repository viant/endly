package webdriver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/viant/endly"
)

// TestLiveBrowserFlow is opt-in so the default unit suite remains hermetic.
// Set ENDLY_WEBDRIVER_REMOTE to an already-running ChromeDriver endpoint.
// ENDLY_CHROME_DEBUGGER_ADDRESS may identify an existing debug-enabled Chrome.
func TestLiveBrowserFlow(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_REMOTE")
	debuggerAddress := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	directCDP := remote == "" && debuggerAddress != ""
	if remote == "" && debuggerAddress == "" {
		t.Skip("ENDLY_WEBDRIVER_REMOTE or ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
<label for="email">Email</label><input id="email" data-testid="email">
<button data-testid="submit" onclick="fetch('/api/orders',{method:'POST'}).then(()=>document.querySelector('#status').textContent='Signed in')">Submit</button>
<div id="status">Pending</div>
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
	if _, err := service.openSession(ctx, request); err != nil {
		t.Fatal(err)
	}
	autoStart := false
	runRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands: []interface{}{
			`page.goto("` + page.URL + `")`,
			`page.getByLabel("Email").fill("qa@example.test")`,
			`page.getByTestId("submit").click()`,
			`createResponse = page.waitForResponse("/api/orders", 201, 5000)`,
			`expect("#status").toHaveText("Signed in", 5000)`,
		},
		Navigation: &NavigationOptions{TimeoutMs: 10_000},
	}
	if err := runRequest.Init(); err != nil {
		t.Fatal(err)
	}
	response, err := service.run(ctx, runRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Assertions) != 1 || !response.Assertions[0].Passed {
		t.Fatalf("assertions=%#v", response.Assertions)
	}
	if response.Data["createResponse"] == nil {
		t.Fatalf("browser response was not captured: %#v", response.Data)
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
