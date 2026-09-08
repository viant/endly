package webdriver

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/viant/endly"
)

func hasResponseWait(actions []*Action) bool {
	for _, action := range actions {
		for _, call := range action.Calls {
			if call != nil && call.Method == "WaitForResponse" {
				return true
			}
		}
	}
	return false
}

// ensureNetworkCaptureLocked prepares capture before the action that triggers
// a request. The caller owns the session command lock.
func (s *service) ensureNetworkCaptureLocked(session *Session) error {
	if session == nil || session.driver == nil {
		return fmt.Errorf("webdriver session not open")
	}
	if !isChromeLike(session.Browser) {
		return fmt.Errorf("page.waitForResponse currently requires Chrome")
	}
	if session.Capture == nil || !session.Capture.Enabled() {
		enableConsole := false
		enableNetwork := true
		includeBodies := false
		session.Capture = newCaptureState(&CaptureStartRequest{
			EnableConsole: &enableConsole,
			EnableNetwork: &enableNetwork,
			IncludeBodies: &includeBodies,
			MaxEntries:    10_000,
		})
	} else {
		session.Capture.mux.Lock()
		session.Capture.enableNetwork = true
		session.Capture.mux.Unlock()
	}
	if session.Remote == "" && session.Backend != "cdp" {
		host, port := pair(session.SessionID)
		session.Remote = fmt.Sprintf("http://%s:%s/wd/hub", host, port)
	}
	_, _ = executeSessionCDP(session, "Network.enable", map[string]any{})
	return nil
}

func (s *service) waitForResponse(context *endly.Context, session *Session, pattern string, status, timeoutMs, afterSequence int) (*NetworkTransaction, error) {
	if timeoutMs <= 0 {
		timeoutMs = 10_000
	}
	match, err := responseURLMatcher(pattern)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		if err := context.Background().Err(); err != nil {
			return nil, err
		}
		session.Capture.Drain(session)
		_, network := session.Capture.Snapshot(10_000, false, true)
		for index := len(network) - 1; index >= 0; index-- {
			transaction := network[index]
			if transaction == nil || transaction.Sequence <= afterSequence || !match(transaction.URL) || status > 0 && transaction.Status != status {
				continue
			}
			return transaction, nil
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("timed out after %dms waiting for browser response URL %q status %d", timeoutMs, pattern, status)
		}
		remaining := time.Until(deadline)
		poll := 100 * time.Millisecond
		if remaining < poll {
			poll = remaining
		}
		if err := waitForContext(context.Background(), poll); err != nil {
			return nil, err
		}
	}
}

func responseURLMatcher(pattern string) (func(string) bool, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("response URL pattern was empty")
	}
	if len(pattern) >= 2 && strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		compiled, err := regexp.Compile(pattern[1 : len(pattern)-1])
		if err != nil {
			return nil, fmt.Errorf("invalid response URL regexp %q: %w", pattern, err)
		}
		return compiled.MatchString, nil
	}
	return func(URL string) bool { return strings.Contains(URL, pattern) }, nil
}
