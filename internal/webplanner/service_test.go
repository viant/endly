package webplanner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlannerDefaultsAreLoopbackAndAuthenticated(t *testing.T) {
	service := NewService(&Config{Port: 8082})
	if service.Config.Host != "127.0.0.1" {
		t.Fatalf("host=%q", service.Config.Host)
	}
	if len(service.Config.Token) < 32 {
		t.Fatalf("token was not generated")
	}
}

func TestAllowedOrigin(t *testing.T) {
	testCases := []struct {
		origin string
		host   string
		allow  bool
	}{
		{"", "127.0.0.1:8082", true},
		{"http://localhost:8082", "127.0.0.1:8082", true},
		{"http://127.0.0.1:3000", "localhost:8082", true},
		{"https://example.test", "127.0.0.1:8082", false},
		{"not a URL", "127.0.0.1:8082", false},
	}
	for _, testCase := range testCases {
		if actual := isAllowedOrigin(testCase.origin, testCase.host); actual != testCase.allow {
			t.Fatalf("origin=%q host=%q allow=%v, wanted %v", testCase.origin, testCase.host, actual, testCase.allow)
		}
	}
}

func TestPlannerTokenAuthorization(t *testing.T) {
	service := NewService(&Config{Token: "secret"})
	request := httptest.NewRequest("POST", "http://127.0.0.1/event?token=secret", nil)
	if !service.authorize(request) {
		t.Fatal("expected authorized request")
	}
	request = httptest.NewRequest("POST", "http://127.0.0.1/event?token=wrong", nil)
	if service.authorize(request) {
		t.Fatal("unexpected authorization")
	}
}

func TestEventEndpointRequiresTokenButAllowsAuthorizedRecorderOrigin(t *testing.T) {
	service := NewService(&Config{Token: "secret"})
	payload := `{"type":"navigation","url":"https://example.test"}`
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/event", strings.NewReader(payload))
	request.Header.Set("Origin", "https://example.test")
	response := httptest.NewRecorder()
	service.handleEvent(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d, wanted 403", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "http://127.0.0.1/event?token=secret", strings.NewReader(payload))
	request.Header.Set("Origin", "https://example.test")
	response = httptest.NewRecorder()
	service.handleEvent(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "https://example.test" {
		t.Fatalf("origin header=%q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestProcessEventRecordsDSL(t *testing.T) {
	service := NewService(&Config{Token: "secret"})
	payload := `{
		"type":"click",
		"targetTag":"BUTTON",
		"targetHTML":"<button id=\"save\">Save</button>",
		"holderHTML":"<div><button id=\"save\">Save</button></div>",
		"url":"https://example.test/edit",
		"timestamp":123
	}`
	request := httptest.NewRequest("POST", "http://127.0.0.1/event", strings.NewReader(payload))
	if err := service.processEvent(request); err != nil {
		t.Fatal(err)
	}
	recorded := service.recordedActions()
	if len(recorded) != 1 {
		t.Fatalf("recorded=%d, wanted 1", len(recorded))
	}
	if !strings.Contains(recorded[0].Expression, ".click()") {
		t.Fatalf("expression=%q", recorded[0].Expression)
	}
}

func TestProcessEventRedactsPassword(t *testing.T) {
	service := NewService(&Config{Token: "secret"})
	payload := `{
		"type":"input",
		"targetTag":"INPUT",
		"targetHTML":"<input id=\"password\" type=\"password\">",
		"holderHTML":"<form><input id=\"password\" type=\"password\"></form>",
		"valueRedacted":true,
		"url":"https://example.test/login"
	}`
	request := httptest.NewRequest("POST", "http://127.0.0.1/event", strings.NewReader(payload))
	if err := service.processEvent(request); err != nil {
		t.Fatal(err)
	}
	recorded := service.recordedActions()
	if len(recorded) != 1 || !recorded[0].ValueRedacted || !strings.Contains(recorded[0].Expression, "$PASSWORD") {
		t.Fatalf("recorded=%#v", recorded)
	}
}
