package endly

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"strings"
	"sync"
)

//AppName represents endly application name
const AppName = "endly - End To End Functional Testing "

//AppVersion represents endly application version
const AppVersion = "0.0.1"

//Manager represnets a workflow manager
type Manager interface {

	//Name returns an application Id
	Name() string

	//Version returns an application version
	Version() string

	//Service return a workflow service for provided Id or error
	Service(name string) (Service, error)

	//Register register service in this manager
	Register(service Service)

	//NewContext returns new workflow context.
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
	return nil, fmt.Errorf("failed to lookup service: '%v' in [%v]", name, strings.Join(available, ","))
}

func (s *manager) Register(service Service) {
	s.services[service.Id()] = service
}

func (s *manager) NewContext(ctx toolbox.Context) *Context {
	sessionID := uuid.NewV1()
	var workflowStack Workflows = make([]*Workflow, 0)
	var result = &Context{
		SessionID: sessionID.String(),
		Context:   ctx,
		Events: &Events{
			mutex:  &sync.Mutex{},
			Events: make([]*Event, 0),
		},
		Workflows: &workflowStack,
	}
	result.Put(serviceManagerKey, s)
	return result
}

//NewManager returns a new manager.
func NewManager() Manager {
	var result = &manager{
		name:     AppName,
		version:  AppVersion,
		services: make(map[string]Service),
	}

	result.Register(NewExecService())
	result.Register(NewTransferService())
	result.Register(NewDeploymentService())
	result.Register(NewHTTPpRunnerService())
	result.Register(NewRestService())
	result.Register(NewProcessService())
	result.Register(NewDaemonService())
	result.Register(NewValidatorService())
	result.Register(NewWorkflowService())
	result.Register(NewVersionControlService())
	result.Register(NewSystemJdkService())
	result.Register(NewBuildService())
	result.Register(NewDockerService())
	result.Register(NewDataStoreUnitService())
	result.Register(NewNopService())
	result.Register(NewLogValidatorService())
	result.Register(NewEventReporterService())
	result.Register(NewNetworkService())
	result.Register(NewSeleniumService())
	return result
}


//Services returns manager services or error
func Services(mgr interface{}) map[string]Service {
	var manager, ok = mgr.(*manager);
	if !ok {
		return nil
	}
	return manager.services
}