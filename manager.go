package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

const AppName = "endly - End To End Functional Testing "
const AppVersion = "0.0.1"

type Manager interface {
	Name() string

	Version() string

	Service(name string) (Service, error)

	Register(service Service)

	CredentialFile(name string) (string, error)

	RegisterCredentialFile(name, file string)

	NewContext(context toolbox.Context) *Context
}

type manager struct {
	name            string
	version         string
	services        map[string]Service
	credentialFiles map[string]string
}

func (s *manager) Name() string {
	return s.name
}

func (s *manager) Version() string {
	return s.version
}

func (s *manager) Service(name string) (Service, error) {
	if result, found := s.services[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup app: %v", name)
}

func (s *manager) Register(service Service) {
	s.services[service.Id()] = service
}

func (s *manager) CredentialFile(name string) (string, error) {
	if result, found := s.credentialFiles[name]; found {
		return result, nil
	}
	return "", fmt.Errorf("Failed to lookup credential: %v", name)
}

func (s *manager) RegisterCredentialFile(name, file string) {
	s.credentialFiles[name] = file
}

func (s *manager) NewContext(ctx toolbox.Context) *Context {
	var result = &Context{
		Context: ctx,
	}
	result.Put(serviceManagerKey, s)
	return result
}


func NewManager() Manager {
	var result = &manager{
		name:            AppName,
		version:         AppVersion,
		services:        make(map[string]Service),
		credentialFiles: make(map[string]string),
	}
	result.Register(NewExecService())
	result.Register(NewTransferService())
	result.Register(NewDeploymentService())
	result.Register(NewScriptService())
	result.Register(NewHttpRunnerService())
	result.Register(NewProcessService())
	result.Register(NewSystemService())
	result.Register(NewWorkflowService())
	result.Register(NewVersionControlService())
	result.Register(NewJdkService())
	result.Register(NewBuildService())
	result.Register(NewDockerService())
	return result
}
