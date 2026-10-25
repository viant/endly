package webdriver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

type fakeCDPServer struct {
	server         *httptest.Server
	mu             sync.Mutex
	URL            string
	title          string
	stopped        bool
	readyState     string
	navigated      chan struct{}
	inputEvents    int
	clickObserved  bool
	keyEvents      []map[string]interface{}
	insertedTexts  []string
	viewportWidth  int
	viewportHeight int
	viewportParams map[string]interface{}
	foreground     bool
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
			case "Page.bringToFront":
				result.mu.Lock()
				result.foreground = true
				result.mu.Unlock()
			case "Emulation.setDeviceMetricsOverride":
				result.mu.Lock()
				result.viewportWidth = int(command.Params["width"].(float64))
				result.viewportHeight = int(command.Params["height"].(float64))
				result.viewportParams = command.Params
				result.mu.Unlock()
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
				case strings.Contains(expression, "click target did not settle"):
					if result.foreground {
						value, kind = map[string]interface{}{"x": 12, "y": 34}, "object"
					}
				case strings.Contains(expression, "getAnimations"):
					value = false
				case strings.Contains(expression, "return {x,y,hit:"):
					value, kind = map[string]interface{}{"x": 10, "y": 20, "hit": true}, "object"
				case strings.Contains(expression, "this.addEventListener('click'"):
					result.clickObserved = false
				case strings.Contains(expression, "const seen=window[key]===true"):
					value = result.clickObserved
					result.clickObserved = false
				case strings.Contains(expression, "document.readyState"):
					value, kind = result.readyState, "string"
				case strings.Contains(expression, "document.title"):
					value, kind = result.title, "string"
				case strings.Contains(expression, "window.location.href"):
					value, kind = result.URL, "string"
				case strings.Contains(expression, "window.innerWidth") && strings.Contains(expression, "window.innerHeight"):
					value, kind = map[string]interface{}{"width": result.viewportWidth, "height": result.viewportHeight}, "object"
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
				if command.Method == "Input.dispatchMouseEvent" && command.Params["type"] == "mouseReleased" {
					result.clickObserved = true
				}
				if command.Method == "Input.dispatchKeyEvent" {
					result.keyEvents = append(result.keyEvents, command.Params)
				}
				if command.Method == "Input.insertText" {
					text, _ := command.Params["text"].(string)
					result.insertedTexts = append(result.insertedTexts, text)
				}
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

func TestDirectCDPBackspaceUsesNativeEditingKeyCode(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Quit()
	element, err := driver.FindElement("css selector", "#submit")
	if err != nil {
		t.Fatal(err)
	}
	if err = element.SendKeys(selenium.BackspaceKey); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.keyEvents) != 2 {
		t.Fatalf("expected keyDown/keyUp, got %v", fake.keyEvents)
	}
	for i, kind := range []string{"keyDown", "keyUp"} {
		event := fake.keyEvents[i]
		if event["type"] != kind || event["key"] != "Backspace" || event["windowsVirtualKeyCode"] != float64(8) {
			t.Fatalf("native Backspace event missing: %v", event)
		}
	}
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

func TestDirectCDPSetViewportSize(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	require.NoError(t, err)
	defer driver.Quit()
	require.Error(t, driver.SetViewportSize(0, 720))
	require.Error(t, driver.SetViewportSize(1280, 16385))
	require.NoError(t, driver.SetViewportSize(1280, 720))
	fake.mu.Lock()
	defer fake.mu.Unlock()
	require.Equal(t, 1280, fake.viewportWidth)
	require.Equal(t, 720, fake.viewportHeight)
	require.Equal(t, float64(1), fake.viewportParams["deviceScaleFactor"])
	require.Equal(t, false, fake.viewportParams["mobile"])
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
			`page.setViewportSize(1280, 720)`,
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
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.viewportWidth != 1280 || fake.viewportHeight != 720 {
		t.Fatalf("viewport=%dx%d, wanted 1280x720", fake.viewportWidth, fake.viewportHeight)
	}
}

func TestDirectCDPNativeEditingKeys(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	require.NoError(t, err)
	defer driver.Quit()
	element, err := driver.FindElement("css selector", "#field")
	require.NoError(t, err)
	for _, key := range []string{selenium.BackspaceKey, selenium.DeleteKey, selenium.LeftArrowKey, selenium.EnterKey, selenium.SpaceKey} {
		require.NoError(t, element.SendKeys(key))
	}
	fake.mu.Lock()
	events := append([]map[string]interface{}(nil), fake.keyEvents...)
	fake.mu.Unlock()
	require.Len(t, events, 10)
	for i := 0; i < len(events); i += 2 {
		down, up := events[i], events[i+1]
		require.Equal(t, "keyDown", down["type"])
		require.Equal(t, "keyUp", up["type"])
		require.NotEmpty(t, down["code"])
		require.NotZero(t, down["windowsVirtualKeyCode"])
		require.Equal(t, down["code"], up["code"])
		require.Equal(t, down["windowsVirtualKeyCode"], up["windowsVirtualKeyCode"])
		require.NotContains(t, up, "text", "keyUp must not insert another character")
	}
	require.Equal(t, float64(8), events[0]["windowsVirtualKeyCode"])
	require.Equal(t, float64(46), events[2]["windowsVirtualKeyCode"])
	require.Equal(t, "\r", events[6]["text"])
	require.Equal(t, " ", events[8]["text"])
	document := "profile:\n  id: customer-intent\n"
	require.NoError(t, element.SendKeys(document))
	fake.mu.Lock()
	defer fake.mu.Unlock()
	require.Equal(t, []string{document}, fake.insertedTexts, "multiline text remains one native insertText transaction")
	require.Len(t, fake.keyEvents, 10)
}

func TestDirectClickGeometryForegroundsTarget(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Quit()
	element, err := driver.FindElement(selenium.ByCSSSelector, "#submit")
	if err != nil {
		t.Fatal(err)
	}
	x, y, err := element.(*directCDPElement).center()
	if err != nil || x != 12 || y != 34 {
		t.Fatalf("background geometry: %v %v %v", x, y, err)
	}
}
