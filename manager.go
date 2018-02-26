package endly

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

//AppName represents endly application name
const AppName = "endly"

//EndlyNamespace represents endly namespace
const EndlyNamespace = "github.com/viant/endly/"

//Service represnets a workflow manager
type Manager interface {
	//Name returns an application ID
	Name() string

	//Version returns an application version
	Version() string

	//Service return a workflow service for provided ID, request,  or error
	Service(input interface{}) (Service, error)

	//Register register service in this manager
	Register(service Service)

	//NewContext returns new workflow context.
	NewContext(context toolbox.Context) *Context

	Run(context *Context, request interface{}) (interface{}, error)
}

type manager struct {
	name                 string
	version              string
	serviceByID          map[string]Service
	serviceByRequestType map[reflect.Type]Service
}

func (s *manager) Name() string {
	return s.name
}

func (s *manager) Version() string {
	return s.version
}

func (s *manager) Service(input interface{}) (Service, error) {
	if serviceId, ok := input.(string); ok {
		if result, found := s.serviceByID[serviceId]; found {
			return result, nil
		}
	} else if toolbox.IsStruct(input) {
		if result, found := s.serviceByRequestType[reflect.TypeOf(input)]; found {
			return result, nil
		}
	}
	var available = toolbox.MapKeysToStringSlice(s.serviceByID)
	return nil, fmt.Errorf("failed to lookup service: '%v' in [%v]", input, strings.Join(available, ","))
}

func (s *manager) Register(service Service) {
	s.serviceByID[service.ID()] = service
	for _, action := range service.Actions() {
		if actionRoute, err := service.ServiceActionRoute(action); err == nil {
			request := actionRoute.RequestProvider()
			s.serviceByRequestType[reflect.TypeOf(request)] = service
		}
	}
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
		name:                 AppName,
		version:              GetVersion(),
		serviceByID:          make(map[string]Service),
		serviceByRequestType: make(map[reflect.Type]Service),
	}

	result.Register(NewExecService())

	var storageService = NewStorageService()
	result.Register(storageService)
	//backward compatibility name of service
	result.serviceByID[TransferServiceID] = storageService
	result.Register(NewDeploymentService())
	result.Register(NewHTTPpRunnerService())
	result.Register(NewRestService())
	result.Register(NewProcessService())
	result.Register(NewDaemonService())
	result.Register(NewValidatorService())
	result.Register(NewWorkflowService())
	result.Register(NewVersionControlService())
	result.Register(NewSdkService())
	result.Register(NewBuildService())
	result.Register(NewDockerService())
	result.Register(NewDataStoreUnitService())
	result.Register(NewNopService())
	//backward compatibility name of service
	var loggerService = NewLogValidatorService()
	result.serviceByID[LogServiceID] = loggerService
	result.Register(loggerService)

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

//Run action for supplied request, returns service action response or error
func (m *manager) Run(context *Context, request interface{}) (interface{}, error) {
	manager := NewManager()

	if !toolbox.IsStruct(request) {
		return nil, fmt.Errorf("expected request but had %T", request)
	}
	service, err := manager.Service(request)
	if err != nil {
		return nil, err
	}
	if context == nil {
		context = manager.NewContext(toolbox.NewContext())
		defer context.Close()
	}
	response := service.Run(context, request)
	return response.Response, response.err
}

//Services returns manager serviceByID or error
func Services(mgr interface{}) map[string]Service {
	var manager, ok = mgr.(*manager)
	if !ok {
		return nil
	}
	return manager.serviceByID
}

//GetVersion return endly version
func GetVersion() string {
	resource := url.NewResource(fmt.Sprintf("mem://%v/Version", EndlyNamespace))
	version, _ := resource.DownloadText()
	return version
}
