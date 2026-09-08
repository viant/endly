package webdriver

import (
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
	"sync"
)

// Session represents a selenium session
type Session struct {
	SessionID         string
	Browser           string
	Pid               int
	Server            string
	Remote            string
	CDPRemote         string
	DriverSessionID   string
	Capture           *CaptureState
	Net               *netTracker
	driver            selenium.WebDriver
	service           *selenium.Service
	Capabilities      []string
	PageLoadStrategy  string
	Attached          bool
	Backend           string
	mu                sync.Mutex
	cleanupRegistered bool
}

func (s *Session) Driver() selenium.WebDriver {
	return s.driver
}

func (s *Session) Close() {
	driver := s.driver
	s.driver = nil
	s.CDPRemote = ""
	s.DriverSessionID = ""
	service := s.service
	s.service = nil
	if driver != nil {
		_ = driver.Quit()
	}
	if service != nil {
		_ = service.Stop()
	}
}

// SeleniumSessions reprents selenium sessions.
type sessions struct {
	mu       sync.RWMutex
	Sessions map[string]*Session
}

var sessionKey = (*sessions)(nil)

var sessionStoreInit sync.Mutex

// Sessions returns a point-in-time snapshot. Session objects remain live, but
// callers cannot race with additions/removals by mutating the backing map.
func Sessions(context *endly.Context) map[string]*Session {
	store := getSessionStore(context)
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := make(map[string]*Session, len(store.Sessions))
	for key, session := range store.Sessions {
		result[key] = session
	}
	return result
}

func getSessionStore(context *endly.Context) *sessions {
	sessionStoreInit.Lock()
	defer sessionStoreInit.Unlock()
	var result *sessions
	if !context.Contains(sessionKey) {
		result = &sessions{
			Sessions: make(map[string]*Session),
		}
		context.Put(sessionKey, result)
	}
	context.GetInto(sessionKey, &result)
	return result
}

func lookupSession(context *endly.Context, sessionID string) (*Session, bool) {
	store := getSessionStore(context)
	store.mu.RLock()
	defer store.mu.RUnlock()
	result, ok := store.Sessions[sessionID]
	return result, ok
}

func putSession(context *endly.Context, sessionID string, session *Session) {
	store := getSessionStore(context)
	store.mu.Lock()
	store.Sessions[sessionID] = session
	store.mu.Unlock()
}
