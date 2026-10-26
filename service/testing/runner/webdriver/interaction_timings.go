package webdriver

import (
	"sync"
	"time"
)

// InteractionTiming is an inclusive wall span. Nested spans must not be summed.
// Phase names come from code, never a selector, script, credential or DOM value.
type InteractionTiming struct {
	Phase      string
	StartedMs  float64
	DurationMs float64
}

type interactionTimings struct {
	mu      sync.Mutex
	clock   func() time.Time
	started time.Time
	spans   []*InteractionTiming
}

func newInteractionTimings(clock func() time.Time) *interactionTimings {
	return &interactionTimings{clock: clock, started: clock()}
}

var noInteractionTiming = func() {}

func startInteractionPhase(t *interactionTimings, phase string) func() {
	if t == nil {
		return noInteractionTiming
	}
	started := t.clock()
	return func() {
		ended := t.clock()
		span := &InteractionTiming{Phase: phase, StartedMs: float64(started.Sub(t.started)) / float64(time.Millisecond), DurationMs: float64(ended.Sub(started)) / float64(time.Millisecond)}
		t.mu.Lock()
		t.spans = append(t.spans, span)
		t.mu.Unlock()
	}
}

func timedInteraction(t *interactionTimings, phase string, run func() error) error {
	end := startInteractionPhase(t, phase)
	defer end()
	return run()
}

func (t *interactionTimings) snapshot() []*InteractionTiming {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*InteractionTiming, len(t.spans))
	for i, span := range t.spans {
		copied := *span
		out[i] = &copied
	}
	return out
}

func cdpTimingPhase(method string) string {
	switch method {
	case "Page.bringToFront", "Input.dispatchMouseEvent", "Input.dispatchKeyEvent", "Input.insertText", "DOM.setFileInputFiles", "Emulation.setDeviceMetricsOverride":
		return "cdp." + method
	default:
		return "cdp.other"
	}
}
