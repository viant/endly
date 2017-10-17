package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"time"
	"github.com/viant/toolbox"
	"sync"
	"strings"
)

type ServiceResponse struct {
	Status   string
	Error    string
	Response interface{}
}

type Service interface {
	Id() string
	State() common.Map
	Run(context *Context, request interface{}) *ServiceResponse
	NewRequest(action string) (interface{}, error)
	Mutex() *sync.RWMutex
}

type AbstractService struct {
	Service
	id    string
	state common.Map
	mutex *sync.RWMutex
}

func Pairs(params ... interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	for i := 0; i < len(params); i += 2 {
		var key = toolbox.AsString(params[i])
		result[key] = params[i+1]
	}
	return result
}

func (s *AbstractService) AddEvent(context *Context, eventType string, value map[string]interface{}, level ... int) *Event {
	if len(level) == 0 {
		level = []int{Info}
	}
	var workflow = context.CurrentWorkflow()
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
		task, _ = state.Get("task").(*WorkflowTask)
	}
	var event = &Event{
		Timestamp: time.Now(),
		Workflow:  workflowName,
		Task:      task,
		Activity:  activity,
		Type:      eventType,
		Level:     level[0],
		Value:     value,
	}
	context.Events.Push(event)
	return event
}

func (s *AbstractService) Begin(context *Context, source interface{}, value map[string]interface{}, level ... int) *Event {
	if len(level) == 0 {
		level = []int{Debug}
	}
	var simpleTypeName = toolbox.DereferenceType(source).Name()
	lastDotPosition := strings.LastIndex(simpleTypeName, ".")
	if lastDotPosition != -1 {
		simpleTypeName = string(simpleTypeName[lastDotPosition:])
	}
	var eventType = fmt.Sprintf("%v.Start", simpleTypeName)
	event := s.AddEvent(context, eventType, value, level...)
	return event
}

func (s *AbstractService) End(context *Context) func(*Event, map[string]interface{}) *Event {
	return func(startEvent *Event, value map[string]interface{}) *Event {
		var eventType = strings.Replace(startEvent.Type, ".Start", ".End", 1)
		event := s.AddEvent(context, eventType, value, startEvent.Level)
		event.StartEvent = startEvent
		event.TimeTakenMs = int((event.Timestamp.UnixNano() - startEvent.Timestamp.UnixNano()) / int64(time.Millisecond))
		return event
	}
}

func (s *AbstractService) Id() string {
	return s.id
}

func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.mutex
}

func (s *AbstractService) State() common.Map {
	return s.state
}

func (s *AbstractService) NewRequest(action string) (interface{}, error) {
	return nil, fmt.Errorf("Unsupported action: %v", action)
}

func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:    id,
		mutex: &sync.RWMutex{},
		state: common.NewMap(),
	}
}
