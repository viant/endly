package webdriver

import (
	"errors"
	"testing"

	"github.com/tebeka/selenium"
	"github.com/viant/assertly"
	"github.com/viant/endly/service/testing/validator"
)

func TestNavigation_WithDefaults(t *testing.T) {
	nav := navigationWithDefaults(nil)
	if nav.TimeoutMs <= 0 {
		t.Fatalf("expected TimeoutMs default")
	}
	if nav.ScrollDelayMs <= 0 || nav.StableWindowMs <= 0 || nav.MaxScrollSteps <= 0 {
		t.Fatalf("expected defaults: %#v", nav)
	}
	if nav.MaxScrollGrowthPx <= 0 {
		t.Fatalf("expected bounded scroll growth: %#v", nav)
	}
}

func TestPageMetrics_AtBottom(t *testing.T) {
	if !(pageMetrics{Y: 900, Viewport: 100, Height: 1000}).AtBottom() {
		t.Fatal("expected bottom")
	}
	if (pageMetrics{Y: 100, Viewport: 100, Height: 1000}).AtBottom() {
		t.Fatal("did not expect bottom")
	}
}

func TestRunResponse_Assertion(t *testing.T) {
	validation := assertly.NewValidation()
	validation.PassedCount = 2
	response := &RunResponse{Assert: &validator.AssertResponse{Validation: validation}}
	assertions := response.Assertion()
	if len(assertions) != 1 || assertions[0].PassedCount != 2 {
		t.Fatalf("unexpected assertions: %#v", assertions)
	}
}

func TestAttachDefaultsToChrome(t *testing.T) {
	request := &OpenSessionRequest{DebuggerAddress: "127.0.0.1:9222"}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	if request.Browser != ChromeBrowser {
		t.Fatalf("browser=%q, wanted chrome", request.Browser)
	}
}

func TestOpenSessionRejectsInvalidPageLoadStrategy(t *testing.T) {
	request := &OpenSessionRequest{PageLoadStrategy: "eventually"}
	if err := request.Init(); err == nil {
		t.Fatal("expected invalid page-load strategy error")
	}
}

func TestStopLoadingRouteIsRegistered(t *testing.T) {
	service := New()
	if _, err := service.Route("stop-loading"); err != nil {
		t.Fatal(err)
	}
}

func TestMatchesExpectation(t *testing.T) {
	testCases := []struct {
		name        string
		actual      interface{}
		expectation *CallExpectation
		matched     bool
	}{
		{"normalized text", " Ready\n now ", &CallExpectation{Matcher: "equal", Value: "Ready now"}, true},
		{"contains text", "Order is ready", &CallExpectation{Matcher: "contains", Value: "is ready"}, true},
		{"regexp", "Order 42", &CallExpectation{Matcher: "equal", Value: `/Order \d+/`}, true},
		{"boolean", true, &CallExpectation{Matcher: "equal", Value: true}, true},
		{"mismatch", "pending", &CallExpectation{Matcher: "equal", Value: "ready"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matched, _, err := matchesExpectation([]interface{}{testCase.actual, nil}, testCase.expectation)
			if err != nil {
				t.Fatal(err)
			}
			if matched != testCase.matched {
				t.Fatalf("matched=%v, wanted %v", matched, testCase.matched)
			}
		})
	}
}

func TestNavigation_IsPageLoadTimeout(t *testing.T) {
	if !isPageLoadTimeout(errors.New("timeout")) {
		t.Fatalf("expected timeout match")
	}
	if !isPageLoadTimeout(&selenium.Error{LegacyCode: 21, Message: "timeout"}) {
		t.Fatalf("expected legacy timeout match")
	}
	if isPageLoadTimeout(errors.New("other")) {
		t.Fatalf("did not expect match")
	}
}

func TestNetTracker_Inflight(t *testing.T) {
	tracker := &netTracker{}
	_ = tracker.consume("Network.requestWillBeSent", nil)
	_ = tracker.consume("Network.requestWillBeSent", nil)
	if tracker.Inflight() != 2 {
		t.Fatalf("expected inflight 2, got %d", tracker.Inflight())
	}
	_ = tracker.consume("Network.loadingFinished", nil)
	if tracker.Inflight() != 1 {
		t.Fatalf("expected inflight 1, got %d", tracker.Inflight())
	}
	_ = tracker.consume("Network.loadingFailed", nil)
	if tracker.Inflight() != 0 {
		t.Fatalf("expected inflight 0, got %d", tracker.Inflight())
	}
	_ = tracker.consume("Network.loadingFailed", nil) // shouldn't go negative
	if tracker.Inflight() != 0 {
		t.Fatalf("expected inflight 0, got %d", tracker.Inflight())
	}
}
