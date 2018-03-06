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

//Manager represents a workflow manager
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

func (m *manager) Name() string {
	return m.name
}

func (m *manager) Version() string {
	return m.version
}

//Service returns service for supplied request or name.
func (m *manager) Service(input interface{}) (Service, error) {
	if serviceID, ok := input.(string); ok {
		if result, found := m.serviceByID[serviceID]; found {
			return result, nil
		}
	} else if toolbox.IsStruct(input) {
		if result, found := m.serviceByRequestType[reflect.TypeOf(input)]; found {
			return result, nil
		}
	}
	var available = toolbox.MapKeysToStringSlice(m.serviceByID)
	return nil, fmt.Errorf("failed to lookup service: '%v' in [%v]", input, strings.Join(available, ","))
}

func (m *manager) Register(service Service) {
	m.serviceByID[service.ID()] = service
	for _, action := range service.Actions() {
		if actionRoute, err := service.ServiceActionRoute(action); err == nil {
			request := actionRoute.RequestProvider()
			m.serviceByRequestType[reflect.TypeOf(request)] = service
		}
	}
}

func (m *manager) NewContext(ctx toolbox.Context) *Context {
	sessionID := toolbox.AsString(time.Now().Unix())
	if UUID, err := uuid.NewV1(); err == nil {
		sessionID = UUID.String()
	}
	var workflowStack Workflows = make([]*WorkflowRun, 0)
	var result = &Context{
		SessionID: sessionID,
		Context:   ctx,
		Wait:      &sync.WaitGroup{},
		Workflows: &workflowStack,
	}
	_ = result.Put(serviceManagerKey, m)
	return result
}

//New returns a new manager.
func New() Manager {
	var result = &manager{
		name:                 AppName,
		version:              GetVersion(),
		serviceByID:          make(map[string]Service),
		serviceByRequestType: make(map[reflect.Type]Service),
	}

	result.Register(newNopService())
	for _, provider := range *Registry {
		result.Register(provider())
	}
	return result
}

//Run runs action for supplied request, returns service action response or error
func (m *manager) Run(context *Context, request interface{}) (interface{}, error) {
	manager := New()

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

//GetVersion returns endly version
func GetVersion() string {
	resource := url.NewResource(fmt.Sprintf("mem://%v/Version", Namespace))
	version, _ := resource.DownloadText()
	return version
}
