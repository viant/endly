package mobile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		case "POST /session/session-1/element":
			value = map[string]interface{}{"element-6066-11e4-a52e-4f735466cecf": "element-1"}
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

func TestExecutorReturnsExpectationFailureAsValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		value := interface{}(map[string]interface{}{"sessionId": "session-1"})
		if r.URL.Path == "/session/session-1/element" {
			value = map[string]interface{}{"element-6066-11e4-a52e-4f735466cecf": "element-1"}
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
