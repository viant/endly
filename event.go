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
	//Error loging level
	Error
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
	Workflow    string                   //workflow Id
	Task        *WorkflowTask            //task
	Activity    *WorkflowServiceActivity //activity details
	Level       int                      //logging level
	Type        string                   //event type
	Value       map[string]interface{}   //event value
}

//Info returns basic event info
func (e *Event) Info() string {
	var name = ""
	if value, ok := e.Value["Id"]; ok {
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
