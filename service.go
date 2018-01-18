package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
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

	//NewRequest creates a new supported request for this service for the supplied action.
	NewRequest(action string) (interface{}, error)

	//Mutex to sync access to the state if needed.
	Mutex() *sync.RWMutex

	Actions() []string
}

//ServiceResponse service response
type ServiceResponse struct {
	Status string
	Error  string

	Response interface{}
}

//AbstractService represenst an abstract service.
type AbstractService struct {
	Service
	actions []string
	id      string
	state   data.Map
	mutex   *sync.RWMutex
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

//Validate checks if supplied source can be validated and if so checks it.
func (s *AbstractService) Validate(source interface{}, response *ServiceResponse) error {
	if validator, ok := source.(Validateable); ok {
		err := validator.Validate()
		if err != nil {
			response.Status = "error"
			response.Error = fmt.Sprintf("failed to validate %T, %v", source, err)
			return err
		}
	}
	return nil
}

//Mutex returns a mutex
func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.mutex
}

//State returns this service state map.
func (s *AbstractService) State() data.Map {
	return s.state
}

//NewRequest returns error for supplied action
func (s *AbstractService) NewRequest(action string) (interface{}, error) {
	return nil, fmt.Errorf("unsupported action: %v", action)
}

//NewAbstractService creates a new abstract service.
func NewAbstractService(id string, actions ...string) *AbstractService {
	return &AbstractService{
		id:      id,
		actions: actions,
		mutex:   &sync.RWMutex{},
		state:   data.NewMap(),
	}
}

//Validateable represets validator interface
type Validateable interface {
	Validate() error
}
