package manager

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Option func(*Service)

func WithMaxSessions(limit int) Option {
	return func(service *Service) {
		if limit > 0 {
			service.maxSessions = limit
		}
	}
}

func WithSessionTTL(ttl time.Duration) Option {
	return func(service *Service) {
		if ttl > 0 {
			service.sessionTTL = ttl
		}
	}
}

func WithOperationRetention(retention time.Duration) Option {
	return func(service *Service) {
		if retention > 0 {
			service.operationRetention = retention
		}
	}
}

func WithMaxEvents(limit int) Option {
	return func(service *Service) {
		if limit > 0 {
			service.maxEvents = limit
		}
	}
}

func WithCallbackHosts(hosts ...string) Option {
	return func(service *Service) {
		for _, host := range hosts {
			host = strings.ToLower(strings.TrimSpace(host))
			if host != "" {
				service.callbackHosts[host] = true
			}
		}
	}
}

func WithPrivateCallbacks(enabled bool) Option {
	return func(service *Service) {
		service.allowPrivateCallbacks = enabled
	}
}

func WithCallbackHTTPClient(client *http.Client) Option {
	return func(service *Service) {
		if client != nil {
			service.callbackClient = client
		}
	}
}

func WithAllowedActions(actions ...string) Option {
	return func(service *Service) {
		for _, group := range actions {
			for _, action := range strings.Split(group, ",") {
				action = strings.TrimSpace(action)
				if action != "" {
					service.allowedActions[action] = true
				}
			}
		}
	}
}

func (s *Service) StartJanitor(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.cleanup(now.UTC())
			}
		}
	}()
}

func (s *Service) validateCallbacks(callbacks []Callback) error {
	for _, callback := range callbacks {
		parsed, err := url.Parse(callback.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			return &ValidationError{Message: "callback URL must use http or https"}
		}
		host := strings.ToLower(parsed.Hostname())
		if !s.callbackHosts[host] {
			return &ValidationError{Message: "callback host was not allowed: " + host}
		}
		if ip := net.ParseIP(host); !s.allowPrivateCallbacks && ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()) {
			return &ValidationError{Message: "private callback addresses are not allowed"}
		}
	}
	return nil
}

func (s *Service) cleanup(now time.Time) {
	s.mu.RLock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.mu.RUnlock()
	for _, session := range sessions {
		session.mu.Lock()
		for id, operation := range session.operations {
			if operation.FinishedAt != nil && now.Sub(*operation.FinishedAt) > s.operationRetention {
				delete(session.operations, id)
			}
		}
		expired := now.Sub(session.lastAccess) > s.sessionTTL
		id := session.info.SessionID
		session.mu.Unlock()
		if expired {
			_ = s.Close(context.Background(), id)
		}
	}
}

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
