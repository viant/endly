package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
	"sync"
	"github.com/satori/go.uuid"
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
	sessionId := uuid.NewV1()
	var workflowStack = make([]*Workflow, 0)
	var result = &Context{
		SessionId: sessionId.String(),
		Context:   ctx,
		Events: &Events{
			mutex:  &sync.Mutex{},
			Events: make([]*Event, 0),
		},
		workflowStack: &workflowStack,
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
	result.Register(NewEventReporterService())
	return result
}
