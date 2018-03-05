package exec

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
)

//GetReplayService return replay service
func GetReplayService(basedir string) (ssh.Service, error) {
	fileName, _, _ := toolbox.CallerInfo(3)
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

func openTestContext(manager endly.Manager, target *url.Resource, commandDirectory string, service ssh.Service) (*endly.Context, error) {
	var err error
	context := manager.NewContext(toolbox.NewContext())
	request := &OpenSessionRequest{
		Target:          target,
		CommandsBasedir: commandDirectory,
		ReplayService:   service,
	}
	srv, err := manager.Service(ServiceID)
	if err != nil {
		return nil, err
	}
	response := srv.Run(context, request)
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	context.TerminalSessions()[":"] = context.TerminalSessions()["127.0.0.1:22"]
	return context, nil
}

//OpenTestRecorderContext open recorder context (to capture SSH command)
func OpenTestRecorderContext(manager endly.Manager, target *url.Resource, commandDirectory string) (*endly.Context, error) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	commandDirectory = path.Join(parent, commandDirectory)
	return openTestContext(manager, target, commandDirectory, nil)
}

//OpenTestContext opens test context with SSH commands to replay
func OpenTestContext(manager endly.Manager, target *url.Resource, service ssh.Service) (*endly.Context, error) {
	return openTestContext(manager, target, "", service)
}
