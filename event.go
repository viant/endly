package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"sync"
	"time"
)

const (
	All            = iota //event logging level all
	Error                 //event logging level error
	Info                  //event logging level info
	Debug                 //event logging level debug
	ErrorEventType = "Error"
	SleepEventType = "Sleep"
)

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
	Workflow    string                   //workflow name
	Task        *WorkflowTask            //task
	Activity    *WorkflowServiceActivity //activity details
	Level       int                      //logging level
	Type        string                   //event type
	Value       map[string]interface{}   //event value
}


//Info returns basic event info
func (e *Event) Info() string {
	var name = ""
	if value, ok := e.Value["name"]; ok {
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
