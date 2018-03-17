package exec

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
)

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
	if _, ok := context.TerminalSessions()[":"]; !ok {
		context.TerminalSessions()[":"] = context.TerminalSessions()[response.SessionID]
	}
	return nil
}

//NewSSHRecodingContext open recorder context (to capture SSH command)
func NewSSHRecodingContext(manager endly.Manager, target *url.Resource, sessionDir string) (*endly.Context, error) {
	return NewSSHMultiRecordingContext(manager, map[string]*url.Resource{
		sessionDir: target,
	})
}

//NewSSHMultiRecordingContext open multi recorded session
func NewSSHMultiRecordingContext(manager endly.Manager, sessions map[string]*url.Resource) (*endly.Context, error) {
	context := manager.NewContext(toolbox.NewContext())
	fileName, _, _ := toolbox.CallerInfo(3)
	parent, _ := path.Split(fileName)
	for baseDir, target := range sessions {
		endly.LogF("Recoding %v: in %v\n", target.Host(), baseDir)
		baseDir = path.Join(parent, baseDir)
		if err := openSSHSession(context, target, baseDir, nil); err != nil {
			return nil, err
		}
	}
	return context, nil

}

//GetReplayService return replay service
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

//NewSSHReplayContext opens test context with SSH commands to replay
func NewSSHReplayContext(manager endly.Manager, target *url.Resource, basedir string) (*endly.Context, error) {
	return NewSSHMultiReplayContext(manager, map[string]*url.Resource{
		basedir: target,
	})
}

//OpenMultiSessionTestContext opens test context with multi SSH replay/mocks session
func NewSSHMultiReplayContext(manager endly.Manager, sessions map[string]*url.Resource) (*endly.Context, error) {
	context := manager.NewContext(nil)
	for baseDir, target := range sessions {
		endly.LogF("Replaying %v: from %v\n", target.Host(), baseDir)
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
