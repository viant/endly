package webdriver

import (
	"fmt"

	"github.com/viant/endly"
)

// stopLoading deliberately bypasses the per-session command mutex so it can
// interrupt a Chrome navigation currently blocking in WebDriver.Get.
func (s *service) stopLoading(context *endly.Context, request *StopLoadingRequest) (*StopLoadingResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "localhost:4444"
	}
	session, err := s.session(context, sessionID)
	if err != nil {
		return nil, err
	}
	if !isChromeLike(session.Browser) {
		return nil, fmt.Errorf("stop-loading requires an open Chrome session")
	}
	if direct, ok := session.driver.(interface{ StopLoading() error }); ok {
		if err := direct.StopLoading(); err != nil {
			return nil, err
		}
		return &StopLoadingResponse{SessionID: sessionID, Stopped: true}, nil
	}
	remote := session.CDPRemote
	if remote == "" {
		host, port := pair(session.SessionID)
		remote = fmt.Sprintf("http://%s:%s/wd/hub", host, port)
	}
	webdriverSession := session.DriverSessionID
	if webdriverSession == "" {
		return nil, fmt.Errorf("webdriver session ID was empty")
	}
	if _, err := cdpExecute(remote, webdriverSession, "Page.stopLoading", map[string]any{}); err != nil {
		return nil, fmt.Errorf("stop Chrome loading: %w", err)
	}
	return &StopLoadingResponse{SessionID: sessionID, Stopped: true}, nil
}
