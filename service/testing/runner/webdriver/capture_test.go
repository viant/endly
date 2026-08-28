package webdriver

import "testing"

func TestCapture_ParsePerformanceLogMessage_Object(t *testing.T) {
	raw := `{"message":{"method":"Network.requestWillBeSent","params":{"requestId":"1","request":{"url":"https://example.com","method":"GET","headers":{"Authorization":"secret","X":"1"}}}}}`
	method, params, err := parsePerformanceLogMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "Network.requestWillBeSent" {
		t.Fatalf("unexpected method: %s", method)
	}
	if len(params) == 0 {
		t.Fatalf("expected params")
	}
}

func TestCapture_ParsePerformanceLogMessage_String(t *testing.T) {
	raw := `{"message":"{\"method\":\"Network.loadingFinished\",\"params\":{\"requestId\":\"1\",\"timestamp\":123.4}}"}`
	method, params, err := parsePerformanceLogMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "Network.loadingFinished" {
		t.Fatalf("unexpected method: %s", method)
	}
	if len(params) == 0 {
		t.Fatalf("expected params")
	}
}

func TestCapture_RedactHeaders(t *testing.T) {
	headers := map[string]any{
		"Authorization": "Bearer abc",
		"X":             "1",
		"Cookie":        "a=b",
	}
	redactHeaders := map[string]bool{"authorization": true, "cookie": true}
	out := redactIfNeeded(headers, true, redactHeaders)
	if out["Authorization"] != "<redacted>" {
		t.Fatalf("expected redacted Authorization, got: %v", out["Authorization"])
	}
	if out["Cookie"] != "<redacted>" {
		t.Fatalf("expected redacted Cookie, got: %v", out["Cookie"])
	}
	if out["X"] != "1" {
		t.Fatalf("expected X preserved, got: %v", out["X"])
	}
}

func TestCapture_CapBody(t *testing.T) {
	body := "1234567890"
	c := capBody(body, false, 4)
	if c.Data != "1234" || !c.Truncated {
		t.Fatalf("unexpected cap: %#v", c)
	}
}

func TestCapture_URLIncludes(t *testing.T) {
	state := newCaptureState(&CaptureStartRequest{URLIncludes: []string{"/v1/", "api.example.test"}})
	if !state.includesURL("http://127.0.0.1:4198/v1/polly/guide") {
		t.Fatal("expected /v1/ URL to be captured")
	}
	if !state.includesURL("https://api.example.test/query") {
		t.Fatal("expected named API host to be captured")
	}
	if state.includesURL("http://127.0.0.1:4198/@vite/client") {
		t.Fatal("expected Vite module URL to be filtered")
	}
}
