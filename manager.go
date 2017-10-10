package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

const AppName = "endly - End To End Functional Testing "
const AppVersion = "0.0.1"

type Manager interface {
	Name() string

	Version() string

	Service(name string) (Service, error)

	Register(service Service)

	NewContext(context toolbox.Context) *Context
}

type manager struct {
	name     string
	version  string
	services map[string]Service
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
	var available = toolbox.MapKeysToStringSlice(s.services)
	return nil, fmt.Errorf("Failed to lookup service: '%v' in [%v]", name, strings.Join(available, ","))
}

func (s *manager) Register(service Service) {
	s.services[service.Id()] = service
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
		name:     AppName,
		version:  AppVersion,
		services: make(map[string]Service),
	}
	result.Register(NewExecService())
	result.Register(NewTransferService())
	result.Register(NewDeploymentService())
	result.Register(NewScriptService())
	result.Register(NewHttpRunnerService())
	result.Register(NewProcessService())
	result.Register(NewSystemService())
	result.Register(NewValidatorService())
	result.Register(NewWorkflowService())
	result.Register(NewVersionControlService())
	result.Register(NewJdkService())
	result.Register(NewBuildService())
	result.Register(NewDockerService())
	result.Register(NewDataStoreUnitService())
	result.Register(NewNopService())
	result.Register(NewLogValidatorService())
	return result
}
