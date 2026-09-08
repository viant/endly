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

func containsAll(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}
