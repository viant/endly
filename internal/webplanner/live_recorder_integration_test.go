package webplanner

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/viant/endly/service/testing/runner/webdriver"
)

// TestLiveRecorderDirectCDP is opt-in because it controls an external Chrome.
func TestLiveRecorderDirectCDP(t *testing.T) {
	debuggerAddress := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	if debuggerAddress == "" {
		t.Skip("ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(writer, `<!doctype html><html><body><button data-testid="record">Record me</button></body></html>`)
	}))
	defer page.Close()

	service := New(&Config{
		Host:            "127.0.0.1",
		Port:            1, // force the authenticated HTTP path to fail; console fallback must carry the event
		DebuggerAddress: debuggerAddress,
		DirectCDP:       true,
		Token:           "integration-token",
	})
	defer service.context.Close()
	if err := service.EnsureWebDriver(); err != nil {
		t.Fatal(err)
	}
	if err := service.EnsureSession(); err != nil {
		t.Fatal(err)
	}
	if err := service.startActivityCapture(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RunCommands([]string{`page.goto("` + page.URL + `")`}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RunCommands([]string{`page.getByTestId("record").click()`}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, action := range service.recordedActions() {
			if action != nil && action.Method == "click" && strings.Contains(strings.ToLower(action.Expression), "record") {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	actions := service.recordedActions()
	actionValues := make([]Action, 0, len(actions))
	for _, action := range actions {
		if action != nil {
			actionValues = append(actionValues, *action)
		}
	}
	includeConsole, includeNetwork := true, false
	captured, _ := service.manager.Run(service.context, &webdriver.CaptureExportRequest{MaxEntries: 100, IncludeConsole: &includeConsole, IncludeNetwork: &includeNetwork})
	console := make([]webdriver.ConsoleEntry, 0)
	if response, ok := captured.(*webdriver.CaptureExportResponse); ok {
		for _, entry := range response.Console {
			if entry != nil {
				console = append(console, *entry)
			}
		}
	}
	t.Fatalf("recorded actions=%+v console=%+v", actionValues, console)
}
