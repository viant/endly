package endly

import (
	"github.com/viant/toolbox"
	"fmt"
	"sync"
)

const AppName = "endly - End To End Functional Testing "
const AppVersion = "0.0.1"

type ServiceManager interface {
	Name() string

	Version() string

	Service(name string) (Service, error)

	Register(service Service)

	CredentialFile(name string) (string, error)

	RegisterCredentialFile(name, file string)

	NewContext(context toolbox.Context) *Context
}

type serviceManager struct {
	name            string
	version         string
	services        map[string]Service
	credentialFiles map[string]string
}

func (s *serviceManager) Name() string {
	return s.name
}

func (s *serviceManager) Version() string {
	return s.version
}

func (s *serviceManager) Service(name string) (Service, error) {
	if result, found := s.services[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup app: %v", name)
}

func (s *serviceManager) Register(service Service) {
	s.services[service.Id()] = service
}

func (s *serviceManager) CredentialFile(name string) (string, error) {
	if result, found := s.credentialFiles[name]; found {
		return result, nil
	}
	return "", fmt.Errorf("Failed to lookup credential: %v", name)
}

func (s *serviceManager) RegisterCredentialFile(name, file string) {
	s.credentialFiles[name] = file
}

func (s *serviceManager) NewContext(ctx toolbox.Context) *Context {
	var result = &Context{
		Context: ctx,
	}
	result.Put(serviceManagerKey, s)
	return result
}

var _serviceManager ServiceManager
var _serviceManagerMux = &sync.Mutex{}

func NewServiceManager() (ServiceManager) {
	if _serviceManager != nil {
		return _serviceManager
	}
	_serviceManagerMux.Lock()
	defer _serviceManagerMux.Unlock()

	if _serviceManager != nil {
		return _serviceManager
	}
	_serviceManager = &serviceManager{
		name:            AppName,
		version:         AppVersion,
		services:        make(map[string]Service),
		credentialFiles: make(map[string]string),
	}
	_serviceManager.Register(NewExecService())
	_serviceManager.Register(NewTransferService())
	_serviceManager.Register(NewDeploymentService())
	_serviceManager.Register(NewScriptService())
	return _serviceManager
}
