package webdriver

import (
	"errors"
	"testing"

	"github.com/tebeka/selenium"
)

func TestNavigation_WithDefaults(t *testing.T) {
	nav := navigationWithDefaults(nil)
	if nav.TimeoutMs <= 0 {
		t.Fatalf("expected TimeoutMs default")
	}
	if nav.ScrollDelayMs <= 0 || nav.StableWindowMs <= 0 || nav.MaxScrollSteps <= 0 {
		t.Fatalf("expected defaults: %#v", nav)
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
