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

const (
	messageStyleGeneric = iota
	messageStyleSuccess
	messageStyleError
	messageStyleInput
	messageStyleOutput
)

//Message represent event output message
type Message struct {
	Text  string
	Style int
}

//NewMessage creates a new message
func NewMessage(text string, style int) *Message {
	return &Message{
		Text:text,
		Style:style,
	}
}

//TagMessage represent a typed message, message is align to left and takes most of the space, tag is align to right takes little space.
type TagMessage struct {
	Message    *Message
	Tag        *Message
	IsRepeated bool //flag to reuse the same line if possible, i.e SleepTime
}



//Reporter represents actual event value that can be reported by CLI or Web workflow runner.
type Reporter interface {
	//Returns true if can report for supplied filter
	CanReport(filter map[string]bool) bool

	//Returns zero or more tag messages
	TagMessages() []*TagMessage

	//Returns zero or more messages
	Messages() []*Message
}

//Event represents a workflow event wrapper
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
	var eventType = fmt.Sprintf("%T", e.Value)
	eventType = strings.Replace(eventType, "*", "", len(eventType))
	var fragments = strings.Split(eventType, ".")
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


//AbstractReporter represents an abstract event reporter
type AbstractReporter struct {
	serviceKey string
	eventKeys  []string
	tagMessages []*TagMessage
	message []*Message
}

func (r *AbstractReporter) CanReport(filter map[string]bool) bool {
	if len(filter) == 0 {
		return false
	}
	if len(r.eventKeys) > 0 {
		for _, candidate := range r.eventKeys {
			if allowed, has := filter[candidate]; has {
				return allowed
			}
		}
	}
	if allowed, has := filter[r.serviceKey]; has {
		return allowed
	}
	return false
}

func (r *AbstractReporter) TagMessages() []*TagMessage {
	return r.tagMessages
}

func (r *AbstractReporter) Messages() []*Message {
	return r.message
}

//NewReporter creates a new abstract reporter
func NewReporter(serviceKey string, eventKeys  []string, tagMessages []*TagMessage, messages ...*Message) Reporter {
	var result = &AbstractReporter{
		serviceKey:strings.ToLower(serviceKey),
		eventKeys:eventKeys,
		tagMessages:tagMessages,
		message:messages,
	}

	for i, eventKey :=range result.eventKeys {
		result.eventKeys[i] = strings.ToLower(eventKey)
	}
	return result
}