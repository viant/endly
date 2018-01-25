package endly

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"strings"
	"sync"
	"time"
)

//AppName represents endly application name
const AppName = "endly"

//EndlyNamespace represents endly namespace
const EndlyNamespace = "github.com/viant/endly/"

//Manager represnets a workflow manager
type Manager interface {

	//Name returns an application ID
	Name() string

	//Version returns an application version
	Version() string

	//Service return a workflow service for provided ID or error
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
	s.services[service.ID()] = service
}

func (s *manager) NewContext(ctx toolbox.Context) *Context {
	sessionID := toolbox.AsString(time.Now().Unix())
	if UUID, err := uuid.NewV1(); err == nil {
		sessionID = UUID.String()
	}
	var workflowStack Workflows = make([]*WorkflowControl, 0)
	var result = &Context{
		SessionID: sessionID,
		Context:   ctx,
		Events: &Events{
			mutex:  &sync.Mutex{},
			Events: make([]*Event, 0),
		},
		Workflows: &workflowStack,
	}
	_ = result.Put(serviceManagerKey, s)
	return result
}

//NewManager returns a new manager.
func NewManager() Manager {
	var result = &manager{
		name:     AppName,
		version:  GetVersion(),
		services: make(map[string]Service),
	}

	result.Register(NewExecService())

	var storageService = NewStorageService()
	result.Register(storageService)
	//backward compatibility name of service
	result.services[TransferServiceID] = storageService
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
	result.Register(NewEc2Service())
	result.Register(NewGceService())
	result.Register(NewHTTPEndpointService())
	result.Register(NewSMTPService())
	result.Register(NewLogService())
	return result
}

//Services returns manager services or error
func Services(mgr interface{}) map[string]Service {
	var manager, ok = mgr.(*manager)
	if !ok {
		return nil
	}
	return manager.services
}

//GetVersion return endly version
func GetVersion() string {
	resource := url.NewResource(fmt.Sprintf("mem://%v/Version", EndlyNamespace))
	version, _ := resource.DownloadText()
	return version
}
