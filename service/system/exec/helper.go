package exec

import (
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/location"
)

var sessionsKey = (*model.Sessions)(nil)

// TerminalSessions returns system sessions
func TerminalSessions(context *endly.Context) model.Sessions {
	var result *model.Sessions

	if !context.Contains(sessionsKey) {
		var sessions model.Sessions = make(map[string]*model.Session)
		result = &sessions
		context.AsyncUnsafeKeys[sessionsKey] = true
		_ = context.Put(sessionsKey, result)
	} else {
		context.GetInto(sessionsKey, &result)
	}
	return *result
}

// SessionID returns session I
func SessionID(context *endly.Context, target *location.Resource) string {
	username := ""
	if cred, _ := context.Secrets.GetCredentials(context.Background(), target.Credentials); cred != nil {
		username = cred.Username
	}
	return username + "@" + target.Hostname()
}

// TerminalSession returns Session for passed in target resource.
func TerminalSession(context *endly.Context, target *location.Resource) (*model.Session, error) {
	sessions := TerminalSessions(context)
	var sessionID = SessionID(context, target)

	if !sessions.Has(sessionID) {
		service, err := context.Service(ServiceID)
		if err != nil {
			return nil, err
		}
		response := service.Run(context, &OpenSessionRequest{
			Target: target,
		})
		if response.Err != nil {
			return nil, response.Err
		}
	}
	return sessions[sessionID], nil
}

// Os returns operating system for provide session
func OperatingSystem(context *endly.Context, sessionName string) *model.OperatingSystem {
	var sessions = TerminalSessions(context)
	if session, has := sessions[sessionName]; has {
		os := model.OperatingSystem{
			OSInfo:      session.OsInfo(),
			HardwareInfo: session.HardwareInfo(),
		}
		return &os
	}
	return nil
}

