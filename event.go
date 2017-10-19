package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"sync"
	"time"
)

const (
	All = iota
	Error
	Info
	Debug
	Finest
	ErrorEventType = "Error"
	SleepEventType = "Sleep"
)

type Events struct {
	Events []*Event
	mutex  *sync.Mutex
}

func (e *Events) Push(event *Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if len(e.Events) == 0 {
		e.Events = make([]*Event, 0)
	}
	e.Events = append(e.Events, event)
}

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

type Event struct {
	StartEvent  *Event
	Timestamp   time.Time
	TimeTakenMs int
	Workflow    string
	Task        *WorkflowTask
	Activity    *WorkflowServiceActivity
	Level       int
	Type        string
	Value       map[string]interface{}
}

func (e *Event) Info() string {
	var name = ""
	if value, ok := e.Value["name"]; ok {
		name = toolbox.AsString(value)
	}
	return fmt.Sprintf("%v", name)
}

func (e *Event) ElapsedInfo() string {
	if e.TimeTakenMs == 0 {
		return ""
	}
	return fmt.Sprintf("%v ms", e.TimeTakenMs)
}
