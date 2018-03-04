package endly

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

//EventListener represents an event listener
type EventListener func(event *Event)

//Event represents a workflow event
type Event struct {
	Timestamp  time.Time
	StartEvent *Event    //starting event
	Activity   *Activity //Activity details
	Value      interface{}
}

//Get returns supplied expected type or nil
func (e *Event) Get(expectedType reflect.Type) interface{} {
	if e.Value == nil {
		return nil
	}
	if reflect.TypeOf(e.Value) == expectedType {
		return e.Value
	}
	if serviceResponse, ok := e.Value.(*ServiceResponse); ok && serviceResponse != nil {
		if reflect.TypeOf(serviceResponse.Response) == expectedType {
			return serviceResponse.Response
		}

	}
	return nil
}

//Type returns event type (simple package and struct name)
func (e *Event) Type() string {
	if e.Value == nil {
		return fmt.Sprintf("%T", e.Value)
	}
	var eventType = reflect.TypeOf(e.Value)
	var fullname = eventType.Name()
	var fragments = strings.Split(fullname, ".")
	if len(fragments) > 2 {
		fragments = fragments[len(fragments)-2:]
	}
	return strings.Join(fragments, "_")
}

//NewEvent creates a new event
func NewEvent(value interface{}) *Event {
	return &Event{
		Timestamp: time.Now(),
		Value:     value,
	}
}

//SleepEvent represents a Sleep
type SleepEvent struct {
	SleepTimeMs int
}

//NewSleepEvent create a new sleep event
func NewSleepEvent(sleepTimeMs int) *SleepEvent {
	return &SleepEvent{SleepTimeMs: sleepTimeMs}
}

//ErrorEvent represents a Sleep
type ErrorEvent struct {
	Error string
}

//NewErrorEvent creates a new error event
func NewErrorEvent(message string) *ErrorEvent {
	return &ErrorEvent{
		Error: message,
	}
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

//AsEventListener adds listener
func (e *Events) AsEventListener() EventListener {
	return func(event *Event) {
		e.Push(event)
	}
}

//Drain removes all events from struct to publish them to context
func (e *Events) Drain(context *Context) {
	for i := 0; i < len(e.Events); i++ {
		event := e.Shift()
		if event == nil {
			return
		}
		context.Publish(event)
	}
}
