package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"sync"
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

var _manager Manager
var _managerMux = &sync.Mutex{}

func NewManager() Manager {
	if _manager != nil {
		return _manager
	}
	_managerMux.Lock()
	defer _managerMux.Unlock()

	if _manager != nil {
		return _manager
	}
	_manager = &manager{
		name:            AppName,
		version:         AppVersion,
		services:        make(map[string]Service),
		credentialFiles: make(map[string]string),
	}
	_manager.Register(NewExecService())
	_manager.Register(NewTransferService())
	_manager.Register(NewDeploymentService())
	_manager.Register(NewScriptService())
	_manager.Register(NewHttpRunnerService())
	_manager.Register(NewProcessService())
	_manager.Register(NewSystemService())

	return _manager
}
