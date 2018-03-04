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

//Namespace represents endly namespace
const Namespace = "github.com/viant/endly/"

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
	var workflowStack Workflows = make([]*Control, 0)
	var result = &Context{
		SessionID: sessionID,
		Context:   ctx,
		Wait:      &sync.WaitGroup{},
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

	result.Register(NewService())
	result.Register(newNopService())
	result.Register(newLoggerService())
	for _, provider := range *Registry {
		result.Register(provider())
	}
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
	return response.Response, response.Err
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
	resource := url.NewResource(fmt.Sprintf("mem://%v/Version", Namespace))
	version, _ := resource.DownloadText()
	return version
}
