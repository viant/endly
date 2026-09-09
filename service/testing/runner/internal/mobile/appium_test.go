package mobile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestAppiumClientSessionAndElementFlow(t *testing.T) {
	var mu sync.Mutex
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /wd/hub/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"ready": true}})
		case "POST /wd/hub/session":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"sessionId": "session-1", "capabilities": map[string]interface{}{"platformName": "Android"}}})
		case "POST /wd/hub/session/session-1/element":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"element-6066-11e4-a52e-4f735466cecf": "element-1"}})
		case "GET /wd/hub/session/session-1/element/element-1/text":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": "Welcome"})
		case "GET /wd/hub/session/session-1/screenshot":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": base64.StdEncoding.EncodeToString([]byte("png"))})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": nil})
		}
	}))
	defer server.Close()

	client, err := NewAppiumClient(server.URL+"/wd/hub", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Status(context.Background()); err != nil {
		t.Fatal(err)
	}
	session, err := client.NewSession(context.Background(), map[string]interface{}{"platformName": "Android"})
	if err != nil {
		t.Fatal(err)
	}
	element, err := session.FindElement(context.Background(), "accessibility id", "greeting")
	if err != nil {
		t.Fatal(err)
	}
	text, err := session.Text(context.Background(), element)
	if err != nil || text != "Welcome" {
		t.Fatalf("unexpected text %q, err=%v", text, err)
	}
	screenshot, err := session.Screenshot(context.Background())
	if err != nil || string(screenshot) != "png" {
		t.Fatalf("unexpected screenshot %q, err=%v", screenshot, err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /wd/hub/status",
		"POST /wd/hub/session",
		"POST /wd/hub/session/session-1/element",
		"GET /wd/hub/session/session-1/element/element-1/text",
		"GET /wd/hub/session/session-1/screenshot",
		"DELETE /wd/hub/session/session-1",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests mismatch\n got: %v\nwant: %v", requests, want)
	}
}

func TestAppiumClientReturnsProtocolError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"error": "no such element", "message": "missing"}})
	}))
	defer server.Close()
	client, err := NewAppiumClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Status(context.Background())
	if err == nil || !containsAll(err.Error(), "no such element", "missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppiumSessionDeviceAndCollectionEndpoints(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		value := interface{}(nil)
		switch r.URL.Path {
		case "/session":
			value = map[string]interface{}{"sessionId": "session-1"}
		case "/session/session-1/elements":
			value = []map[string]interface{}{{W3CElementKey: "one"}, {W3CElementKey: "two"}}
		case "/session/session-1/element/one/rect", "/session/session-1/window/rect":
			value = Rect{X: 1, Y: 2, Width: 30, Height: 40}
		case "/session/session-1/contexts":
			value = []string{"NATIVE_APP", "WEBVIEW_app"}
		case "/session/session-1/context":
			if r.Method == http.MethodGet {
				value = "NATIVE_APP"
			}
		case "/session/session-1/orientation":
			if r.Method == http.MethodGet {
				value = "PORTRAIT"
			}
		case "/session/session-1/alert/text":
			value = "Allow?"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	elements, err := session.FindElements(ctx, "id", "row")
	if err != nil || !reflect.DeepEqual(elements, []string{"one", "two"}) {
		t.Fatalf("unexpected elements=%v err=%v", elements, err)
	}
	if rect, err := session.ElementRect(ctx, "one"); err != nil || rect.Width != 30 {
		t.Fatalf("unexpected rect=%+v err=%v", rect, err)
	}
	if contexts, err := session.Contexts(ctx); err != nil || len(contexts) != 2 {
		t.Fatalf("unexpected contexts=%v err=%v", contexts, err)
	}
	if current, err := session.CurrentContext(ctx); err != nil || current != "NATIVE_APP" {
		t.Fatalf("unexpected context=%q err=%v", current, err)
	}
	if err := session.SetContext(ctx, "WEBVIEW_app"); err != nil {
		t.Fatal(err)
	}
	if orientation, err := session.Orientation(ctx); err != nil || orientation != "PORTRAIT" {
		t.Fatalf("unexpected orientation=%q err=%v", orientation, err)
	}
	if err := session.SetOrientation(ctx, "LANDSCAPE"); err != nil {
		t.Fatal(err)
	}
	if err := session.SetLocation(ctx, 34.1, -118.2, 10); err != nil {
		t.Fatal(err)
	}
	if text, err := session.AlertText(ctx); err != nil || text != "Allow?" {
		t.Fatalf("unexpected alert=%q err=%v", text, err)
	}
	if err := session.AcceptAlert(ctx); err != nil {
		t.Fatal(err)
	}
	if err := session.DismissAlert(ctx); err != nil {
		t.Fatal(err)
	}
	if err := session.Back(ctx); err != nil {
		t.Fatal(err)
	}
	if err := session.HideKeyboard(ctx); err != nil {
		t.Fatal(err)
	}
	if err := session.PointerGesture(ctx, []PointerPoint{{X: 10, Y: 20, Down: true}, {X: 10, Y: 100, DurationMs: 200, Up: true}}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"POST /session/session-1/elements", "GET /session/session-1/element/one/rect",
		"GET /session/session-1/contexts", "POST /session/session-1/context",
		"GET /session/session-1/orientation", "POST /session/session-1/orientation",
		"POST /session/session-1/location", "POST /session/session-1/alert/accept",
		"POST /session/session-1/back", "POST /session/session-1/appium/device/hide_keyboard",
		"POST /session/session-1/actions", "DELETE /session/session-1/actions",
	} {
		if !containsString(requests, expected) {
			t.Fatalf("missing request %q in %v", expected, requests)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAll(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}
