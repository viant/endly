package mobile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutorRunsElementActionsAndExpectations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value interface{}
		switch r.Method + " " + r.URL.Path {
		case "POST /session":
			value = map[string]interface{}{"sessionId": "session-1", "capabilities": map[string]interface{}{}}
		case "POST /session/session-1/elements":
			value = []map[string]interface{}{{"element-6066-11e4-a52e-4f735466cecf": "element-1"}}
		case "GET /session/session-1/element/element-1/text":
			value = "Welcome"
		case "GET /session/session-1/element/element-1/displayed":
			value = true
		default:
			value = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, err := NewAppiumClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.NewSession(context.Background(), map[string]interface{}{"platformName": "Android"})
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{
		Session: session,
		Resolve: func(call Call) (Locator, bool, error) {
			if call.Name != "getByTestId" {
				return Locator{}, false, nil
			}
			value, err := requiredStringArg(call, 0)
			return Locator{Using: "accessibility id", Value: value}, true, err
		},
		ActionTimeout: 50 * time.Millisecond,
		PollInterval:  time.Millisecond,
	}
	result, err := executor.Run(context.Background(), []string{
		`greeting = app.getByTestId("greeting").text()`,
		`expect(app.getByTestId("greeting")).toContainText("Welcome", 20)`,
		`app.getByTestId("continue").tap()`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["greeting"] != "Welcome" {
		t.Fatalf("unexpected data: %+v", result.Data)
	}
	if len(result.Validations) != 1 || result.Validations[0].PassedCount != 1 {
		t.Fatalf("unexpected validations: %+v", result.Validations)
	}
}

func TestExecutorCollectionsAndStrictMatching(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		value := interface{}(nil)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			value = map[string]interface{}{"sessionId": "session-1"}
		case r.Method == http.MethodPost && r.URL.Path == "/session/session-1/elements":
			payload := map[string]interface{}{}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload["value"] == "missing" {
				value = []map[string]interface{}{}
			} else {
				value = []map[string]interface{}{{W3CElementKey: "element-1"}, {W3CElementKey: "element-2"}}
			}
		case strings.HasSuffix(r.URL.Path, "/element/element-1/text"):
			value = "first"
		case strings.HasSuffix(r.URL.Path, "/element/element-2/text"):
			value = "second"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{
		Session: session,
		Resolve: func(call Call) (Locator, bool, error) {
			value, err := requiredStringArg(call, 0)
			return Locator{Using: "accessibility id", Value: value}, true, err
		},
		ActionTimeout: 20 * time.Millisecond,
		PollInterval:  time.Millisecond,
	}
	result, err := executor.Run(context.Background(), []string{
		`first = app.getByTestId("row").first().text()`,
		`second = app.getByTestId("row").nth(1).text()`,
		`last = app.getByTestId("row").last().text()`,
		`count = app.getByTestId("row").count()`,
		`missing = app.getByTestId("missing").exists()`,
		`expect(app.getByTestId("row")).toHaveCount(2, 10)`,
		`expect(app.getByTestId("missing")).toBeHidden(10)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["first"] != "first" || result.Data["second"] != "second" || result.Data["last"] != "second" || result.Data["count"] != 2 || result.Data["missing"] != false {
		t.Fatalf("unexpected collection data: %+v", result.Data)
	}
	if len(result.Validations) != 2 || result.Validations[0].PassedCount != 1 || result.Validations[1].PassedCount != 1 {
		t.Fatalf("unexpected validations: %+v", result.Validations)
	}
	_, err = executor.Run(context.Background(), []string{`app.getByTestId("row").text()`})
	if err == nil || !strings.Contains(err.Error(), "matched 2 elements") {
		t.Fatalf("expected strict match error, got %v", err)
	}
}

func TestExecutorDeviceExpectations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := interface{}(map[string]interface{}{"sessionId": "session-1"})
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{
		Session: session,
		Resolve: func(Call) (Locator, bool, error) { return Locator{}, false, nil },
		ExecuteDevice: func(_ context.Context, _ *AppiumSession, call Call) (interface{}, error) {
			switch strings.ToLower(call.Name) {
			case "orientation":
				return "LANDSCAPE", nil
			case "context":
				return "NATIVE_APP", nil
			case "contexts":
				return []string{"NATIVE_APP", "WEBVIEW_app"}, nil
			case "alerttext":
				return "Allow access?", nil
			}
			return nil, nil
		},
		ActionTimeout: 20 * time.Millisecond,
		PollInterval:  time.Millisecond,
	}
	result, err := executor.Run(context.Background(), []string{
		`contexts = device.contexts()`,
		`expect(device.orientation()).toHaveOrientation("LANDSCAPE", 10)`,
		`expect(device.context()).toHaveContext("NATIVE_APP", 10)`,
		`expect(device.alertText()).toHaveAlert(10)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Validations) != 3 || len(result.Data["contexts"].([]string)) != 2 {
		t.Fatalf("unexpected device result: %+v", result)
	}
}

func TestExecutorW3CPointerGestures(t *testing.T) {
	var actionCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		value := interface{}(nil)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			value = map[string]interface{}{"sessionId": "session-1"}
		case r.Method == http.MethodPost && r.URL.Path == "/session/session-1/elements":
			payload := map[string]interface{}{}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			id := "source"
			if payload["value"] == "target" {
				id = "target"
			}
			value = []map[string]interface{}{{W3CElementKey: id}}
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/rect"):
			if strings.Contains(r.URL.Path, "/target/") {
				value = Rect{X: 200, Y: 300, Width: 40, Height: 40}
			} else {
				value = Rect{X: 10, Y: 20, Width: 100, Height: 200}
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/actions"):
			actionCalls.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{
		Session: session,
		Resolve: func(call Call) (Locator, bool, error) {
			value, err := requiredStringArg(call, 0)
			return Locator{Using: "accessibility id", Value: value}, true, err
		},
		ActionTimeout: 50 * time.Millisecond,
		PollInterval:  time.Millisecond,
	}
	_, err = executor.Run(context.Background(), []string{
		`app.getByTestId("source").doubleTap()`,
		`app.getByTestId("source").longPress(250)`,
		`app.getByTestId("source").swipe("up", 0.5)`,
		`app.getByTestId("source").scroll("down", 2)`,
		`app.getByTestId("source").dragTo("target", 300)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if actionCalls.Load() != 6 {
		t.Fatalf("expected 6 pointer action calls, got %d", actionCalls.Load())
	}
}

func TestExecutorReturnsExpectationFailureAsValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		value := interface{}(map[string]interface{}{"sessionId": "session-1"})
		if r.URL.Path == "/session/session-1/elements" {
			value = []map[string]interface{}{{"element-6066-11e4-a52e-4f735466cecf": "element-1"}}
		} else if r.URL.Path == "/session/session-1/element/element-1/text" {
			value = "Waiting"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	executor := &Executor{
		Session: session,
		Resolve: func(call Call) (Locator, bool, error) {
			value, err := requiredStringArg(call, 0)
			return Locator{Using: "accessibility id", Value: value}, true, err
		},
		ActionTimeout: 10 * time.Millisecond,
		PollInterval:  time.Millisecond,
	}
	result, err := executor.Run(context.Background(), []string{`expect(app.getByTestId("status")).toHaveText("Done", 5)`})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Validations) != 1 || !result.Validations[0].HasFailure() {
		t.Fatalf("expected validation failure: %+v", result.Validations)
	}
}
