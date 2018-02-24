package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"reflect"
	"sync"
	"time"
)

const (
	//All logging level.
	All = iota
	//Fatal logging level
	Fatal
	//Info logging level
	Info
	//Debug logging level
	Debug
)

//SleepEventType represents a Sleep
type SleepEventType struct {
	SleepTimeMs int
}

//ErrorEventType represents a Sleep
type ErrorEventType struct {
	Error string
}

//Events represents sychronized slice of Events
type Events struct {
	Events []*Event
	mutex  *sync.Mutex
}

//Push appends an event to the events slice
func (e *Events) Push(event *Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if len(e.Events) == 0 {
		e.Events = make([]*Event, 0)
	}
	e.Events = append(e.Events, event)
}

//Shift removes the first event from the  events slice
func (e *Events) Shift() *Event {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if len(e.Events) == 0 {
		return nil
	}
	var result = e.Events[0]
	e.Events = e.Events[1:]
	return result
}

//Event represents a workflow event
type Event struct {
	StartEvent  *Event                   //starting event
	Timestamp   time.Time                //start time
	TimeTakenMs int                      //time taken
	Workflow    string                   //workflow ID
	Task        *WorkflowTask            //task
	Activity    *WorkflowServiceActivity //Activity details
	Level       int                      //logging level
	Type        string                   //event type
	Value       map[string]interface{}   //event value
}

//Output returns basic event info
func (e *Event) Output() string {
	var name = ""
	if value, ok := e.Value["ID"]; ok {
		name = toolbox.AsString(value)
	}
	return fmt.Sprintf("%v", name)
}

//ElapsedInfo returns elapsed info if time taken is present
func (e *Event) ElapsedInfo() string {
	if e.TimeTakenMs == 0 {
		return ""
	}
	return fmt.Sprintf("%v ms", e.TimeTakenMs)
}

func (e *Event) filterByType(candidate interface{}, expectedType reflect.Type) interface{} {
	if reflect.TypeOf(candidate) == expectedType {
		return candidate
	}
	switch casted := candidate.(type) {
	case *ServiceResponse:
		if casted.Response != nil {
			result := e.filterByType(casted.Response, expectedType)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func (e *Event) get(expectedType reflect.Type) interface{} {
	for _, value := range e.Value {
		result := e.filterByType(value, expectedType)
		if result != nil {
			return result
		}
	}
	return nil
}

//AddEvent add an event to the current context
func AddEvent(context *Context, eventType interface{}, value map[string]interface{}, level ...int) *Event {
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
	if state.Has("Activity") {
		activity, _ = state.Get("Activity").(*WorkflowServiceActivity)
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
