package webdriver

import (
	"strings"
	"testing"
)

func TestDriverCallRejectsMalformedRequestBeforeSession(t *testing.T) {
	for _, request := range []*WebDriverCallRequest{nil, {}, {Call: &MethodCall{}}, {Call: &MethodCall{Method: " \t"}}} {
		// A nil service/context proves validation precedes session lookup and locking.
		response, err := (*service)(nil).callWebDriver(nil, request)
		if response != nil || err == nil || !strings.Contains(err.Error(), "call.method") {
			t.Fatalf("expected bounded call.method diagnostic, got response=%v error=%v", response, err)
		}
	}
}

func TestDriverCallAcceptsNestedMethodWithoutChangingParameters(t *testing.T) {
	request := &WebDriverCallRequest{Call: &MethodCall{Method: "CDPCommand", Parameters: []interface{}{"Network.getCookies", map[string]interface{}{"urls": []string{"https://example.test"}}}}}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	if request.Call.Method != "CDPCommand" || len(request.Call.Parameters) != 2 {
		t.Fatal("validation changed the browser call")
	}
}
