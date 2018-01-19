package endly_test

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"os"
	"path"
	"time"
)

func GetAbstractService() *endly.AbstractService {
	manager := endly.NewManager()
	nop, _ := manager.Service(endly.NopServiceID)
	nopService := nop.(*endly.NopService)
	return nopService.AbstractService
}

func GetDummyCredential() (string, error) {
	return GetCredential("dummy", os.Getenv("USER"), "***")
}

func GetCredential(name, username, password string) (string, error) {
	var credentialFile = path.Join(os.TempDir(), fmt.Sprintf("%v%v.json", name, time.Now().Hour()))
	authConfig := cred.Config{
		Username: username,
		Password: password,
	}
	err := authConfig.Save(credentialFile)
	return credentialFile, err
}

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
		Target:          target,
		CommandsBasedir: commandDirectory,
		ReplayService:   service,
	}
	srv, err := manager.Service(endly.ExecServiceID)
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

func OpenTestRecorderContext(manager endly.Manager, target *url.Resource, commandDirectory string) (*endly.Context, error) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	commandDirectory = path.Join(parent, commandDirectory)
	return openTestContext(manager, target, commandDirectory, nil)
}

func OpenTestContext(manager endly.Manager, target *url.Resource, service ssh.Service) (*endly.Context, error) {
	return openTestContext(manager, target, "", service)
}
