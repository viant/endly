package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"reflect"
	"sync"
	"time"
)

//Service represents a set of capabilities per supported actions/request.
type Service interface {
	//service id
	ID() string

	//service state map
	State() data.Map

	//Run service action for supported request types.
	Run(context *Context, request interface{}) *ServiceResponse

	//ServiceActionRoute returns service action route
	ServiceActionRoute(action string) (*ServiceActionRoute, error)

	Mutex() *sync.RWMutex

	Actions() []string
}

//Validator represents generic validator
type Validator interface {
	Validate() error
}

//Initializer represents generic initializer
type Initializer interface {
	Init() error
}

//ServiceResponse service response
type ServiceResponse struct {
	Status   string
	Error    string
	Response interface{}
	Err      error
}

//ExampleUseCase represents example use case
type ExampleUseCase struct {
	UseCase string
	Data    string
}

//ActionInfo represent an action info
type ActionInfo struct {
	Description string
	Examples    []*ExampleUseCase
}

//ServiceActionRoute represents service action route
type ServiceActionRoute struct {
	Action           string
	RequestInfo      *ActionInfo
	ResponseInfo     *ActionInfo
	RequestProvider  func() interface{}
	ResponseProvider func() interface{}
	Handler          func(context *Context, request interface{}) (interface{}, error)
}

//AbstractService represenst an abstract service.
type AbstractService struct {
	Service
	*sync.RWMutex
	routeByAction  map[string]*ServiceActionRoute
	routeByRequest map[reflect.Type]*ServiceActionRoute
	actions        []string
	id             string
	state          data.Map
}

//Mutex returns a mutex.
func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.RWMutex
}

//Register register action routes
func (s *AbstractService) Register(routes ...*ServiceActionRoute) {
	for _, route := range routes {
		s.routeByAction[route.Action] = route
		s.routeByRequest[reflect.TypeOf(route.RequestProvider())] = route
		s.actions = append(s.actions, route.Action)
	}
}

func (s *AbstractService) addRouteIfConvertible(request interface{}) *ServiceActionRoute {
	var requestType = reflect.TypeOf(request)
	if requestType != nil {
		for k, v := range s.routeByRequest {
			if requestType.Kind() == reflect.Ptr && requestType.Elem().ConvertibleTo(k.Elem()) {

				s.routeByRequest[requestType] = &ServiceActionRoute{
					Action:           v.Action,
					RequestInfo:      v.RequestInfo,
					ResponseInfo:     v.ResponseInfo,
					RequestProvider:  v.RequestProvider,
					ResponseProvider: v.ResponseProvider,
					Handler: func(context *Context, convertibleRequest interface{}) (interface{}, error) {
						var request = v.RequestProvider()
						var requestValue = reflect.ValueOf(request)
						var convertibleValue = reflect.ValueOf(convertibleRequest)
						requestValue.Elem().Set(convertibleValue.Elem().Convert(k.Elem()))
						return v.Handler(context, request)
					},
				}
				return s.routeByRequest[requestType]
			}
		}
	}
	return nil
}

//Run returns a service action for supplied action
func (s *AbstractService) Run(context *Context, request interface{}) (response *ServiceResponse) {
	response = &ServiceResponse{Status: "ok"}
	startEvent := s.Begin(context, request)
	var err error
	defer func() {
		s.End(context)(startEvent, response.Response)
		if err != nil {
			response.Err = err
			response.Status = "error"
			response.Error = fmt.Sprintf("%v", err)
		}
	}()

	if len(s.routeByRequest) == 0 {
		err = NewError(s.ID(), fmt.Sprintf("%T", request), fmt.Errorf("failed to lookup service route: %T", request))
		return response
	}

	service, ok := s.routeByRequest[reflect.TypeOf(request)]
	if !ok {

		service = s.addRouteIfConvertible(request)
		if service == nil {
			err = NewError(s.ID(), fmt.Sprintf("%T", request), fmt.Errorf("failed to lookup service route: %T", request))
			return response
		}
	}

	if initializer, ok := request.(Initializer); ok {
		if err = initializer.Init(); err != nil {
			err = NewError(s.ID(), service.Action, fmt.Errorf("init %T failed: %v", request, err))
			return response
		}
	}

	if validator, ok := request.(Validator); ok {
		if err = validator.Validate(); err != nil {
			err = NewError(s.ID(), service.Action, fmt.Errorf("validation %T failed: %v", request, err))
			return response
		}
	}

	response.Response, err = service.Handler(context, request)
	if err != nil {
		var previous = err
		err = NewError(s.ID(), service.Action, err)
		if previous != err {
			context.Publish(NewErrorEvent(fmt.Sprintf("%v", err)))
		}
		response.Err = err
	}
	return response
}

//ServiceActionRoute returns a service action route for supplied action
func (s *AbstractService) ServiceActionRoute(action string) (*ServiceActionRoute, error) {
	if len(s.routeByAction) > 0 {
		if result, ok := s.routeByAction[action]; ok {
			return result, nil
		}
		return nil, fmt.Errorf("unknown %v.%v service action", s.id, action)
	}
	return nil, fmt.Errorf("unknown %v.%v service action", s.id, action)
}

//Sleep sleeps for provided time in ms
func (s *AbstractService) Sleep(context *Context, sleepTimeMs int) {
	if sleepTimeMs > 0 {
		context.Publish(NewSleepEvent(sleepTimeMs))
		time.Sleep(time.Millisecond * time.Duration(sleepTimeMs))
	}
}

//GetHostAndSSHPort return host and ssh port
func (s *AbstractService) GetHostAndSSHPort(target *url.Resource) (string, int) {
	if target == nil {
		return "", 0
	}
	port := toolbox.AsInt(target.ParsedURL.Port())
	if port == 0 {
		port = 22
	}
	hostname := target.ParsedURL.Hostname()
	if hostname == "" {
		hostname = "127.0.0.1"
	}
	return hostname, port
}

//Actions returns service actions
func (s *AbstractService) Actions() []string {
	return s.actions
}

//Begin add starting event
func (s *AbstractService) Begin(context *Context, value interface{}) *Event {
	return context.Publish(value)
}

//End adds finishing event.
func (s *AbstractService) End(context *Context) func(startEvent *Event, value interface{}) *Event {
	return func(startEvent *Event, value interface{}) *Event {
		endEvent := context.Publish(value)
		endEvent.StartEvent = startEvent
		return endEvent
	}
}

//ID returns this service id.
func (s *AbstractService) ID() string {
	return s.id
}

//State returns this service state map.
func (s *AbstractService) State() data.Map {
	return s.state
}

//NewAbstractService creates a new abstract service.
func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:             id,
		actions:        make([]string, 0),
		RWMutex:        &sync.RWMutex{},
		state:          data.NewMap(),
		routeByAction:  make(map[string]*ServiceActionRoute),
		routeByRequest: make(map[reflect.Type]*ServiceActionRoute),
	}
}

//NopRequest represent no operation to be deprecated
type NopRequest struct{}

//NopParrotRequest represent parrot request
type NopParrotRequest struct {
	In interface{}
}

//nopService represents no operation nopService (deprecated, use workflow, nop instead)
type nopService struct {
	*AbstractService
}

func (s *nopService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "nop",
		RequestInfo: &ActionInfo{
			Description: "no operation action, helper for separating action.Init as self descriptive steps",
		},
		RequestProvider: func() interface{} {
			return &NopRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*NopRequest); ok {
				return req, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "parrot",
		RequestInfo: &ActionInfo{
			Description: "send back supplied In value",
		},
		RequestProvider: func() interface{} {
			return &NopParrotRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*NopParrotRequest); ok {
				return req.In, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//newNopService creates a new NoOperation nopService.
func newNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService("nop"),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
