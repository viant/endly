package endly_test

import (
	"github.com/viant/toolbox"
	"path"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"errors"
)



func GetReplayService(basedir string) (ssh.Service, error) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
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
	request := &endly.OpenSessionRequest{
		Target:target,
		CommandsBasedir:commandDirectory,
		ReplayService:service,
	}
	srv, err := manager.Service(endly.ExecServiceID)
	if err != nil {
		return nil, err
	}
	response := srv.Run(context, request)
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	return context, nil
}

func OpenTestRecorderContext(manager endly.Manager, target *url.Resource, commandDirectory string) (*endly.Context, error) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	commandDirectory =path.Join(parent, commandDirectory)
	return openTestContext(manager, target, commandDirectory, nil)
}

func OpenTestContext(manager endly.Manager, target *url.Resource,  service ssh.Service) (*endly.Context, error) {
	return openTestContext(manager, target, "", service)
}