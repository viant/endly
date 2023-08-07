package endly

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	_ "github.com/viant/endly/unsafe"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// AbstractService represenst an abstract service.
type AbstractService struct {
	Service
	*sync.RWMutex
	routeByAction  map[string]*Route
	routeByRequest map[reflect.Type]*Route
	actions        []string
	id             string
	state          data.Map
}

// Mutex returns a mutex.
func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.RWMutex
}

// Register register action routes
func (s *AbstractService) Register(routes ...*Route) {
	for _, route := range routes {
		s.routeByAction[route.Action] = route
		s.routeByRequest[reflect.TypeOf(route.RequestProvider())] = route
		s.actions = append(s.actions, route.Action)
	}
}

func (s *AbstractService) addRouteIfConvertible(request interface{}) *Route {
	var requestType = reflect.TypeOf(request)
	if requestType != nil {
		for k, v := range s.routeByRequest {
			if requestType.Kind() == reflect.Ptr && requestType.Elem().ConvertibleTo(k.Elem()) {

				s.routeByRequest[requestType] = &Route{
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

// Run returns a service action for supplied action
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
			context.Publish(msg.NewErrorEvent(fmt.Sprintf("%v", err)))
		}
		response.Err = err
	}
	return response
}

// Route returns a service action route for supplied action
func (s *AbstractService) Route(action string) (*Route, error) {
	if result, ok := s.routeByAction[action]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("unknown %v.%v service action", s.id, action)
}

// Sleep sleeps for provided time in ms
func (s *AbstractService) Sleep(context *Context, sleepTimeMs int) {
	if sleepTimeMs == 0 {
		return
	}
	sleepTime := time.Millisecond * time.Duration(sleepTimeMs)
	if sleepTime < time.Minute {
		if context.IsLoggingEnabled() {
			context.Publish(msg.NewSleepEvent(sleepTimeMs))
		}
		time.Sleep(sleepTime)
		return
	}

	startTime := time.Now()
	for {
		if context.IsLoggingEnabled() {
			context.Publish(msg.NewSleepEvent(1000))
		}
		if time.Now().Sub(startTime) >= sleepTime {
			break
		}
		time.Sleep(time.Second)
	}
}

// GetHostAndSSHPort return host and ssh port
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

// Actions returns service actions
func (s *AbstractService) Actions() []string {
	return s.actions
}

// Begin add starting event
func (s *AbstractService) Begin(context *Context, value interface{}) msg.Event {
	return context.Publish(value)
}

// End adds finishing event.
func (s *AbstractService) End(context *Context) func(startEvent msg.Event, value interface{}) msg.Event {
	return func(startEvent msg.Event, value interface{}) msg.Event {
		return context.PublishWithStartEvent(value, startEvent)
	}
}

// ID returns this service id.
func (s *AbstractService) ID() string {
	return s.id
}

// State returns this service state map.
func (s *AbstractService) State() data.Map {
	return s.state
}

func (s *AbstractService) RunInBackground(context *Context, handler func() error) (err error) {
	wait := &sync.WaitGroup{}
	wait.Add(1)
	var done uint32 = 0
	go func() {
		for {
			if atomic.LoadUint32(&done) == 1 {
				break
			}
			s.Sleep(context, 2000)
		}
	}()

	go func() {
		defer wait.Done()
		err = handler()

	}()
	wait.Wait()
	atomic.StoreUint32(&done, 1)
	return err
}

// NewAbstractService creates a new abstract service.
func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:             id,
		actions:        make([]string, 0),
		RWMutex:        &sync.RWMutex{},
		state:          data.NewMap(),
		routeByAction:  make(map[string]*Route),
		routeByRequest: make(map[reflect.Type]*Route),
	}
}

// NopRequest represent no operation to be deprecated
type NopRequest struct {
	In interface{}
}

// nopService represents no operation nopService (deprecated, use workflow, nop instead)
type nopService struct {
	*AbstractService
}

func (s *nopService) registerRoutes() {
	s.Register(&Route{
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
				return req.In, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// newNopService creates a new NoOperation nopService.
func newNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService("nop"),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
