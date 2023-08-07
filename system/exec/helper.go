package exec

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
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
func SessionID(context *endly.Context, target *url.Resource) string {
	username := ""
	if config, _ := context.Secrets.GetCredentials(target.Credentials); config != nil {
		username = config.Username
	}
	return username + "@" + target.Host()
}

// TerminalSession returns Session for passed in target resource.
func TerminalSession(context *endly.Context, target *url.Resource) (*model.Session, error) {
	sessions := TerminalSessions(context)
	if target == nil {
		return nil, errors.New("target was empty")
	}
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
		return session.Os
	}
	return nil
}

func openSSHSession(context *endly.Context, target *url.Resource, commandDirectory string, service ssh.Service) error {
	request := &OpenSessionRequest{
		Target:        target,
		Basedir:       commandDirectory,
		ReplayService: service,
	}
	response := &OpenSessionResponse{}
	if err := endly.Run(context, request, response); err != nil {
		return err
	}
	if _, ok := TerminalSessions(context)[":"]; !ok {
		TerminalSessions(context)[":"] = TerminalSessions(context)[response.SessionID]
	}
	return nil
}

// NewSSHRecodingContext open recorder context (to capture SSH command)
func NewSSHRecodingContext(manager endly.Manager, target *url.Resource, sessionDir string) (*endly.Context, error) {
	return NewSSHMultiRecordingContext(manager, map[string]*url.Resource{
		sessionDir: target,
	})
}

// NewSSHMultiRecordingContext open multi recorded session
func NewSSHMultiRecordingContext(manager endly.Manager, sessions map[string]*url.Resource) (*endly.Context, error) {
	context := manager.NewContext(toolbox.NewContext())
	fileName, _, _ := toolbox.CallerInfo(4)
	parent, _ := path.Split(fileName)
	for baseDir, target := range sessions {
		baseDir = path.Join(parent, baseDir)
		if err := openSSHSession(context, target, baseDir, nil); err != nil {
			return nil, err
		}
	}
	return context, nil

}

// GetReplayService return replay service
func GetReplayService(basedir string) (ssh.Service, error) {
	fileName, _, _ := toolbox.DiscoverCaller(3, 10, "helper.go")
	parent, _ := path.Split(fileName)
	replayDirectory := path.Join(parent, basedir)
	if !toolbox.FileExists(replayDirectory) {
		return nil, fmt.Errorf("replay directory does not exist: %v", replayDirectory)
	}
	commands, err := ssh.NewReplayCommands(path.Join(parent, basedir))
	if err != nil {
		return nil, err
	}
	err = commands.Load()
	if err != nil {
		return nil, err
	}
	service := ssh.NewReplayService(commands.Shell(), commands.System(), commands, nil)
	return service, nil
}

// NewSSHReplayContext opens test context with SSH commands to replay
func NewSSHReplayContext(manager endly.Manager, target *url.Resource, basedir string) (*endly.Context, error) {
	return NewSSHMultiReplayContext(manager, map[string]*url.Resource{
		basedir: target,
	})
}

// OpenMultiSessionTestContext opens test context with multi SSH replay/mocks session
func NewSSHMultiReplayContext(manager endly.Manager, sessions map[string]*url.Resource) (*endly.Context, error) {
	context := manager.NewContext(nil)
	for baseDir, target := range sessions {
		service, err := GetReplayService(baseDir)
		if err != nil {
			return nil, err
		}
		if err := openSSHSession(context, target, "", service); err != nil {
			return nil, err
		}
	}
	return context, nil
}
