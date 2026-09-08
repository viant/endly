package webdriver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/viant/endly"
)

type fakeCDPServer struct {
	server      *httptest.Server
	mu          sync.Mutex
	URL         string
	title       string
	stopped     bool
	readyState  string
	navigated   chan struct{}
	inputEvents int
}

func newFakeCDPServer(t *testing.T) *fakeCDPServer {
	result := &fakeCDPServer{URL: "https://example.test/", title: "Direct CDP", readyState: "complete", navigated: make(chan struct{}, 4)}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", func(writer http.ResponseWriter, request *http.Request) {
		wsURL := "ws" + strings.TrimPrefix(result.server.URL, "http") + "/devtools/page/one"
		result.mu.Lock()
		title, targetURL := result.title, result.URL
		result.mu.Unlock()
		_ = json.NewEncoder(writer).Encode([]map[string]interface{}{{
			"id": "one", "type": "page", "title": title, "url": targetURL, "webSocketDebuggerUrl": wsURL,
		}})
	})
	mux.HandleFunc("/json/close/one", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("Target is closing"))
	})
	mux.HandleFunc("/devtools/page/one", func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer connection.Close()
		for {
			var command struct {
				ID     int64                  `json:"id"`
				Method string                 `json:"method"`
				Params map[string]interface{} `json:"params"`
			}
			if err := connection.ReadJSON(&command); err != nil {
				return
			}
			response := map[string]interface{}{"id": command.ID, "result": map[string]interface{}{}}
			emitConsole := false
			switch command.Method {
			case "Page.navigate":
				result.mu.Lock()
				result.URL = command.Params["url"].(string)
				result.mu.Unlock()
				select {
				case result.navigated <- struct{}{}:
				default:
				}
			case "Page.stopLoading":
				result.mu.Lock()
				result.stopped = true
				result.mu.Unlock()
			case "Page.captureScreenshot":
				response["result"] = map[string]interface{}{"data": base64.StdEncoding.EncodeToString([]byte("png"))}
			case "Runtime.evaluate":
				expression, _ := command.Params["expression"].(string)
				value := interface{}(true)
				kind := "boolean"
				result.mu.Lock()
				switch {
				case strings.Contains(expression, "document.readyState"):
					value, kind = result.readyState, "string"
				case strings.Contains(expression, "document.title"):
					value, kind = result.title, "string"
				case strings.Contains(expression, "window.location.href"):
					value, kind = result.URL, "string"
				case strings.Contains(expression, "document.documentElement.outerHTML"):
					value, kind = "<html><body>Direct</body></html>", "string"
				case strings.Contains(expression, ").length"):
					value, kind = float64(2), "number"
					if strings.Contains(expression, "#submit") {
						value = float64(1)
					}
				case strings.Contains(expression, "innerText"):
					value, kind = "Direct", "string"
				}
				result.mu.Unlock()
				response["result"] = map[string]interface{}{"result": map[string]interface{}{"type": kind, "value": value}}
			case "Runtime.enable":
				emitConsole = true
			case "Input.dispatchMouseEvent", "Input.dispatchKeyEvent", "Input.insertText":
				result.mu.Lock()
				result.inputEvents++
				result.mu.Unlock()
			}
			if err := connection.WriteJSON(response); err != nil {
				return
			}
			if emitConsole {
				_ = connection.WriteJSON(map[string]interface{}{
					"method": "Runtime.consoleAPICalled",
					"params": map[string]interface{}{"type": "log", "args": []map[string]interface{}{{"value": "direct console"}}},
				})
			}
		}
	})
	result.server = httptest.NewServer(mux)
	t.Cleanup(result.server.Close)
	return result
}

func TestDirectCDPDriver(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Quit()
	title, err := driver.Title()
	if err != nil || title != "Direct CDP" {
		t.Fatalf("title=%q err=%v", title, err)
	}
	if err := driver.Get("https://example.test/next"); err != nil {
		t.Fatal(err)
	}
	URL, err := driver.CurrentURL()
	if err != nil || URL != "https://example.test/next" {
		t.Fatalf("URL=%q err=%v", URL, err)
	}
	elements, err := driver.FindElements("css selector", ".row")
	if err != nil || len(elements) != 2 {
		t.Fatalf("elements=%d err=%v", len(elements), err)
	}
	text, err := elements[0].Text()
	if err != nil || text != "Direct" {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if err := elements[0].Click(); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	inputEvents := fake.inputEvents
	fake.mu.Unlock()
	if inputEvents != 3 {
		t.Fatalf("input events=%d, wanted 3", inputEvents)
	}
	if err := driver.StopLoading(); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	stopped := fake.stopped
	fake.mu.Unlock()
	if !stopped {
		t.Fatal("Page.stopLoading was not received")
	}
	select {
	case <-fake.navigated:
	default:
	}
	fake.mu.Lock()
	fake.readyState = "loading"
	fake.mu.Unlock()
	navigationResult := make(chan error, 1)
	go func() { navigationResult <- driver.Get("https://example.test/stream") }()
	<-fake.navigated
	if err := driver.StopLoading(); err != nil {
		t.Fatal(err)
	}
	if err := <-navigationResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("navigation error=%v, wanted context.Canceled", err)
	}
	screenshot, err := driver.Screenshot()
	if err != nil || string(screenshot) != "png" {
		t.Fatalf("screenshot=%q err=%v", screenshot, err)
	}
	logs, err := driver.Log("browser")
	if err != nil || len(logs) == 0 || !strings.Contains(logs[0].Message, "direct console") {
		t.Fatalf("logs=%#v err=%v", logs, err)
	}
}

func TestRunWithDirectCDPBackend(t *testing.T) {
	fake := newFakeCDPServer(t)
	manager := endly.New()
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	service := New().(*service)
	request := &OpenSessionRequest{DirectCDP: true, DebuggerAddress: fake.server.URL}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	session, err := service.openSession(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if session.Backend != "cdp" || !session.Attached {
		t.Fatalf("session=%#v", session)
	}
	autoStart := false
	runRequest := &RunRequest{
		SessionID: request.SessionID,
		AutoStart: &autoStart,
		Commands: []interface{}{
			`title = page.title()`,
			`page.locator("#submit").click()`,
			`expect(page).toHaveTitle("Direct CDP", 500)`,
		},
	}
	if err := runRequest.Init(); err != nil {
		t.Fatal(err)
	}
	response, err := service.run(ctx, runRequest)
	if err != nil {
		t.Fatal(err)
	}
	if response.Data["title"] != "Direct CDP" || len(response.Assertions) != 1 || !response.Assertions[0].Passed {
		t.Fatalf("response=%#v", response)
	}
}
