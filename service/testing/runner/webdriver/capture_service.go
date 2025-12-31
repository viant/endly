package webdriver

import (
	"fmt"
	"strings"

	"github.com/viant/endly"
)

func (s *service) captureStart(context *endly.Context, request *CaptureStartRequest) (*CaptureStartResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	sess, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.driver == nil {
		return nil, fmt.Errorf("webdriver session not open: %s", sessionID)
	}

	sess.Capture = newCaptureState(request)
	if request.SinkURL != "" {
		if err := sess.Capture.StartSink(s.fs, request.SinkURL, request.FlushIntervalMs); err != nil {
			return nil, err
		}
	}

	warning := ""
	if sess.Remote == "" {
		host, port := pair(sess.SessionID)
		sess.Remote = fmt.Sprintf("http://%v:%v/wd/hub", host, port)
	}

	// Best-effort CDP enable (Chrome/Edge chromedriver only).
	if strings.EqualFold(sess.Browser, ChromeBrowser) {
		wdSession := sess.driver.SessionID()
		if wdSession != "" {
			_, _ = cdpExecute(sess.Remote, wdSession, "Network.enable", map[string]any{})
			_, _ = cdpExecute(sess.Remote, wdSession, "Runtime.enable", map[string]any{})
		}
		// Verify performance logging is enabled; otherwise we won't see events.
		caps, capErr := sess.driver.Capabilities()
		if capErr != nil {
			warning = fmt.Sprintf("capture enabled, but failed to read capabilities: %v", capErr)
		} else if !hasPerformanceLogging(caps) {
			warning = "capture enabled, but performance logging is not enabled for this session; reopen browser session with capture-support"
		}
	} else {
		warning = fmt.Sprintf("capture enabled, but browser %q is not supported (Chrome/Edge only)", sess.Browser)
	}

	return &CaptureStartResponse{
		SessionID: sess.SessionID,
		Enabled:   true,
		Warning:   warning,
	}, nil
}

func (s *service) captureStop(context *endly.Context, request *CaptureStopRequest) (*CaptureStopResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	sess, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Capture != nil {
		sess.Capture.Drain(sess)
		_ = sess.Capture.CloseSink()
	}
	if sess.driver != nil && sess.Remote != "" && strings.EqualFold(sess.Browser, ChromeBrowser) {
		wdSession := sess.driver.SessionID()
		if wdSession != "" {
			_, _ = cdpExecute(sess.Remote, wdSession, "Network.disable", map[string]any{})
			_, _ = cdpExecute(sess.Remote, wdSession, "Runtime.disable", map[string]any{})
		}
	}
	return &CaptureStopResponse{
		SessionID: sess.SessionID,
		Summary:   captureSummary(sess),
	}, nil
}

func (s *service) captureStatus(context *endly.Context, request *CaptureStatusRequest) (*CaptureStatusResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	sess, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	return &CaptureStatusResponse{
		SessionID: sess.SessionID,
		Summary:   captureSummary(sess),
	}, nil
}

func (s *service) captureClear(context *endly.Context, request *CaptureClearRequest) (*CaptureClearResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	sess, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Capture != nil {
		sess.Capture.Clear()
	}
	return &CaptureClearResponse{SessionID: sess.SessionID}, nil
}

func (s *service) captureExport(context *endly.Context, request *CaptureExportRequest) (*CaptureExportResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	sess, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Capture == nil {
		return nil, fmt.Errorf("capture not started for session: %s", sess.SessionID)
	}
	sess.Capture.Drain(sess)

	includeConsole := true
	includeNetwork := true
	if request.IncludeConsole != nil {
		includeConsole = *request.IncludeConsole
	}
	if request.IncludeNetwork != nil {
		includeNetwork = *request.IncludeNetwork
	}
	console, network := sess.Capture.Snapshot(request.MaxEntries, includeConsole, includeNetwork)

	return &CaptureExportResponse{
		SessionID: sess.SessionID,
		Summary:   sess.Capture.Summary(),
		Console:   console,
		Network:   network,
	}, nil
}

func captureSummary(sess *Session) *CaptureSummary {
	if sess == nil || sess.Capture == nil {
		return &CaptureSummary{}
	}
	return sess.Capture.Summary()
}

func hasPerformanceLogging(caps map[string]any) bool {
	// selenium/log uses "goog:loggingPrefs" key.
	for _, key := range []string{"goog:loggingPrefs", "loggingPrefs"} {
		raw, ok := caps[key]
		if !ok || raw == nil {
			continue
		}
		if m, ok := raw.(map[string]any); ok {
			for k := range m {
				if strings.EqualFold(k, "performance") {
					return true
				}
			}
		}
	}
	return false
}
