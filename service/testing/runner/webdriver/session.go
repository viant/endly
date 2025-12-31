package webdriver

import (
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

// Session represents a selenium session
type Session struct {
	SessionID    string
	Browser      string
	Pid          int
	Server       string
	Remote       string
	Capture      *CaptureState
	Net          *netTracker
	driver       selenium.WebDriver
	service      *selenium.Service
	Capabilities []string
}

func (s Session) Driver() selenium.WebDriver {
	return s.driver
}

func (s Session) Close() {
	if s.driver != nil {
		s.driver.Quit()
	}
	if s.service != nil {
		s.service.Stop()
	}
}

// SeleniumSessions reprents selenium sessions.
type sessions struct {
	Sessions map[string]*Session
}

var sessionKey = (*sessions)(nil)

func Sessions(context *endly.Context) map[string]*Session {
	var result *sessions
	if !context.Contains(sessionKey) {
		result = &sessions{
			Sessions: make(map[string]*Session),
		}
		context.Put(sessionKey, result)
	}
	context.GetInto(sessionKey, &result)
	return result.Sessions
}
