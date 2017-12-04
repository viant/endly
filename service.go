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
	Id() string

	//service state map
	State() data.Map

	//Run service action for supported request types.
	Run(context *Context, request interface{}) *ServiceResponse

	//NewRequest creates a new supported request for this service for the supplied action.
	NewRequest(action string) (interface{}, error)

	//Mutex to sync access to the state if needed.
	Mutex() *sync.RWMutex
}

//ServiceResponse service response
type ServiceResponse struct {
	Status   string
	Error    string
	Response interface{}
}

//AbstractService represenst an abstract service.
type AbstractService struct {
	Service
	id    string
	state data.Map
	mutex *sync.RWMutex
}

//Pairs returns map for pairs.
func Pairs(params ...interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	for i := 0; i < len(params); i += 2 {
		var key = toolbox.AsString(params[i])
		result[key] = params[i+1]
	}
	return result
}

//Sleep sleeps for provided time in ms
func (s *workflowService) Sleep(context *Context, sleepTimeMs int) {
	if sleepTimeMs > 0 {
		var sleepEventType = &SleepEventType{SleepTimeMs: sleepTimeMs}
		s.AddEvent(context, sleepEventType, Pairs("value", sleepEventType), Info)
		time.Sleep(time.Millisecond * time.Duration(sleepTimeMs))
	}
}

//AddEvent add event
func (s *AbstractService) AddEvent(context *Context, eventType interface{}, value map[string]interface{}, level ...int) *Event {
	if len(level) == 0 {
		level = []int{Info}
	}
	if !toolbox.IsString(eventType) {
		eventType = getSimpleTypeName(eventType)
	}
	var workflow = context.Workflows.Last()
	var workflowName = ""
	if workflow != nil {
		workflowName = workflow.Name
	}
	state := context.state
	var activity *WorkflowServiceActivity
	var task *WorkflowTask
	if state.Has("activity") {
		activity, _ = state.Get("activity").(*WorkflowServiceActivity)
	}
	if state.Has("task") {
		task, _ = state.Get(":task").(*WorkflowTask)
	}
	var event = &Event{
		Timestamp: time.Now(),
		Workflow:  workflowName,
		Task:      task,
		Activity:  activity,
		Type:      toolbox.AsString(eventType),
		Level:     level[0],
		Value:     value,
	}
	context.Events.Push(event)

	if context.EventLogger != nil {
		err := context.EventLogger.Log(event)
		if err != nil {
			fmt.Printf("failed to log event: %v\n", err)
		}
	}
	return event
}

//Begin add starting event
func (s *AbstractService) Begin(context *Context, source interface{}, value map[string]interface{}, level ...int) *Event {
	if len(level) == 0 {
		level = []int{Debug}
	}
	simpleTypeName := getSimpleTypeName(source)
	var eventType = fmt.Sprintf("%v.Start", simpleTypeName)
	event := s.AddEvent(context, eventType, value, level...)
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
		event := s.AddEvent(context, eventType, value, startEvent.Level)
		event.StartEvent = startEvent
		event.TimeTakenMs = int((event.Timestamp.UnixNano() - startEvent.Timestamp.UnixNano()) / int64(time.Millisecond))
		return event
	}
}

//Id returns this service id.
func (s *AbstractService) Id() string {
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

//NewRequest returns error for supplied action
func (s *AbstractService) NewRequest(action string) (interface{}, error) {
	return nil, fmt.Errorf("Unsupported action: %v", action)
}

//NewAbstractService creates a new abstract service.
func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:    id,
		mutex: &sync.RWMutex{},
		state: data.NewMap(),
	}
}
