package webdriver

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	selog "github.com/tebeka/selenium/log"
)

// netTracker maintains best-effort inflight request counts from Chrome "performance" log entries.
// It is used by the navigation guard to detect "network idle" without requiring capture to be enabled.
type netTracker struct {
	mux        sync.Mutex
	inflight   int
	lastChange time.Time
}

func (t *netTracker) Inflight() int {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.inflight
}

func (t *netTracker) LastChange() time.Time {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.lastChange
}

func (t *netTracker) Drain(driver any) {
	wd, ok := driver.(interface {
		Log(typ selog.Type) ([]selog.Message, error)
	})
	if !ok {
		return
	}
	messages, err := wd.Log(selog.Performance)
	if err != nil {
		return
	}
	for _, msg := range messages {
		method, params, perr := parsePerformanceLogMessage(msg.Message)
		if perr != nil {
			continue
		}
		_ = t.consume(method, params)
	}
}

func (t *netTracker) consume(method string, params json.RawMessage) error {
	switch method {
	case "Network.requestWillBeSent":
		t.bump(+1)
		return nil
	case "Network.loadingFinished", "Network.loadingFailed":
		t.bump(-1)
		return nil
	default:
		return nil
	}
}

func (t *netTracker) bump(delta int) {
	t.mux.Lock()
	defer t.mux.Unlock()
	now := time.Now()
	t.inflight += delta
	if t.inflight < 0 {
		t.inflight = 0
	}
	t.lastChange = now
}

func networkIdle(inflight int, threshold int) bool {
	return inflight <= threshold
}

func formatNetIdle(inflight int, threshold int) string {
	return fmt.Sprintf("inflight=%d threshold=%d", inflight, threshold)
}

func isChromeLike(browser string) bool {
	b := strings.ToLower(strings.TrimSpace(browser))
	return b == "" || b == ChromeBrowser || b == "edge"
}
