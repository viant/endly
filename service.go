package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"reflect"
	"strings"
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

	//Mutex to sync access to the state if needed.
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
	err      error
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
	routeByAction  map[string]*ServiceActionRoute
	routeByRequest map[reflect.Type]*ServiceActionRoute
	actions        []string
	id             string
	state          data.Map
	mutex          *sync.RWMutex
}

//Pairs returns map for pairs.
func Pairs(params ...interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	for i := 0; i+1 < len(params); i += 2 {
		var key = toolbox.AsString(params[i])
		result[key] = params[i+1]
	}
	return result
}

//Register register action routes
func (s *AbstractService) Register(routes ...*ServiceActionRoute) {
	for _, route := range routes {
		s.routeByAction[route.Action] = route
		s.routeByRequest[reflect.TypeOf(route.RequestProvider())] = route
		s.actions = append(s.actions, route.Action)
	}
}

//Run returns a service action for supplied action
func (s *AbstractService) Run(context *Context, request interface{}) (response *ServiceResponse) {
	response = &ServiceResponse{Status: "ok"}
	startEvent := s.Begin(context, request, Pairs("request", request))
	var err error
	defer func() {
		s.End(context)(startEvent, Pairs("response", response))
		if err != nil {
			response.err = err
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
		err = NewError(s.ID(), fmt.Sprintf("%T", request), fmt.Errorf("failed to lookup service route: %T", request))
		return response
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
			var eventType = &ErrorEventType{Error: fmt.Sprintf("%v", err)}
			AddEvent(context, eventType, Pairs("value", eventType), Info)
		}
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
		var sleepEventType = &SleepEventType{SleepTimeMs: sleepTimeMs}
		AddEvent(context, sleepEventType, Pairs("value", sleepEventType), Info)
		time.Sleep(time.Millisecond * time.Duration(sleepTimeMs))
	}
}

//Actions returns service actions
func (s *AbstractService) Actions() []string {
	return s.actions
}

//Begin add starting event
func (s *AbstractService) Begin(context *Context, source interface{}, value map[string]interface{}, level ...int) *Event {
	if len(level) == 0 {
		level = []int{Debug}
	}
	simpleTypeName := getSimpleTypeName(source)
	var eventType = fmt.Sprintf("%v.Start", simpleTypeName)
	event := AddEvent(context, eventType, value, level...)
	return event
}

func getSimpleTypeName(source interface{}) string {
	var simpleTypeName = toolbox.DereferenceType(source).Name()
	lastDotPosition := strings.LastIndex(simpleTypeName, ".")
	if lastDotPosition != -1 {
		simpleTypeName = string(simpleTypeName[lastDotPosition:])
	}
	return simpleTypeName
}

//End adds finishing event.
func (s *AbstractService) End(context *Context) func(*Event, map[string]interface{}) *Event {
	return func(startEvent *Event, value map[string]interface{}) *Event {
		var eventType = strings.Replace(startEvent.Type, ".Start", ".End", 1)
		event := AddEvent(context, eventType, value, startEvent.Level)
		event.StartEvent = startEvent
		event.TimeTakenMs = int((event.Timestamp.UnixNano() - startEvent.Timestamp.UnixNano()) / int64(time.Millisecond))
		return event
	}
}

//ID returns this service id.
func (s *AbstractService) ID() string {
	return s.id
}

//Mutex returns a mutex
func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.mutex
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
		mutex:          &sync.RWMutex{},
		state:          data.NewMap(),
		routeByAction:  make(map[string]*ServiceActionRoute),
		routeByRequest: make(map[reflect.Type]*ServiceActionRoute),
	}
}

//Validateable represets validator interface
type Validateable interface {
	Validate() error
}
