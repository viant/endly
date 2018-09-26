package endly

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

//Run runs action for supplied context request and response. Response has to be pointer or nil
func Run(context *Context, request, result interface{}) error {
	if context == nil {
		manager := New()
		context = manager.NewContext(nil)
		defer context.Close()
	}
	var resultValue reflect.Value
	if result != nil {
		resultValue = reflect.ValueOf(result)
		if resultValue.Kind() != reflect.Ptr {
			return fmt.Errorf("expected result as pointer, but had %T", result)
		}
	}
	manager, err := context.Manager()
	if err != nil {
		return err
	}
	response, err := manager.Run(context, request)
	if err != nil {
		return err
	}
	if serviceResponse, ok := result.(*ServiceResponse); ok {
		serviceResponse.Response = response
		serviceResponse.Status = "ok"
		if err != nil {
			serviceResponse.Status = "error"
			serviceResponse.Err = err
			serviceResponse.Error = err.Error()
		}
		return err
	}
	if result == nil || response == nil {
		return nil
	}
	return toolbox.DefaultConverter.AssignConverted(result, response)
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
		if actionRoute, err := service.Route(action); err == nil {
			request := actionRoute.RequestProvider()
			m.serviceByRequestType[reflect.TypeOf(request)] = service
		}
	}
}

func (m *manager) NewContext(ctx toolbox.Context) *Context {
	if ctx == nil {
		ctx = toolbox.NewContext()
	}
	sessionID := toolbox.AsString(time.Now().Unix())
	if UUID, err := uuid.NewV1(); err == nil {
		sessionID = UUID.String()
	}
	var result = &Context{
		SessionID:       sessionID,
		Context:         ctx,
		Wait:            &sync.WaitGroup{},
		AsyncUnsafeKeys: make(map[interface{}]bool),
		Secrets:         secret.New("", false),
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
	if !toolbox.IsStruct(request) {
		return nil, fmt.Errorf("expected request but had %T", request)
	}
	manager, err := context.Manager()
	if err != nil {
		manager = m
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
