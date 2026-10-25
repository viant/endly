package webdriver

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/endly"
)

func TestResponseWaitRequestBodyMatching(t *testing.T) {
	filter := "\"operation\":\"detail\""
	body := "{\"operation\":\"detail\"}"
	for _, tc := range []struct {
		name string
		body *CapturedBody
		want bool
	}{
		{"plain", &CapturedBody{Data: body}, true},
		{"utf8", &CapturedBody{Encoding: "utf8", Data: body}, true},
		{"base64", &CapturedBody{Encoding: "base64", Data: base64.StdEncoding.EncodeToString([]byte(body))}, true},
		{"other operation", &CapturedBody{Data: "{\"operation\":\"other\"}"}, false},
		{"missing body", nil, false},
		{"truncated", &CapturedBody{Data: body, Truncated: true}, false},
		{"invalid encoding", &CapturedBody{Encoding: "base64", Data: "%%%"}, false},
		{"unknown encoding", &CapturedBody{Encoding: "binary", Data: body}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &NetworkTransaction{RequestBody: tc.body}
			if got := matchesResponseRequestBody(tx, filter); got != tc.want {
				t.Fatalf("match=%v want=%v", got, tc.want)
			}
			if !matchesResponseRequestBody(tx, "") {
				t.Fatal("default URL-only behavior changed")
			}
		})
	}
}

func TestRunResponseWaitDistinguishesSameURLRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"matching", "{\"operation\":\"detail\"}", false},
		{"unrelated", "{\"operation\":\"other\"}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			defer ctx.Close()
			bodies := false
			capture := newCaptureState(&CaptureStartRequest{IncludeBodies: &bodies})
			driver := &runtimeDriver{capture: capture, responseOnGet: &NetworkTransaction{RequestID: "current", URL: "https://example.test/rpc", Status: 200, RequestBody: &CapturedBody{Data: tc.body}}}
			putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, Remote: "http://127.0.0.1:1", driver: driver, Capture: capture})
			filter := "\"operation\":\"detail\""
			request := &RunRequest{SessionID: "fake", Commands: []interface{}{
				"page.goto(\"https://example.test\")",
				fmt.Sprintf("matched = page.waitForResponse(%q, 200, 5, %q)", "/rpc", filter),
			}}
			if err := request.Init(); err != nil {
				t.Fatal(err)
			}
			if !hasResponseBodyWait(request.Actions) {
				t.Fatal("body filter was not retained by parser")
			}
			service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
			response, err := service.run(ctx, request)
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error %v", err)
			}
			if !capture.includeBodies {
				t.Fatal("request body capture was not enabled before navigation")
			}
			if !tc.wantError && response.Data["matched"] == nil {
				t.Fatal("matching transaction missing")
			}
			if tc.wantError && strings.Contains(err.Error(), filter) {
				t.Fatal("body filter echoed in diagnostic")
			}
		})
	}
}
