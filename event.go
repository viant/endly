package endly

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

//EventListener represents an event Listener
type EventListener func(event *Event)

const (
	MessageStyleGeneric = iota
	MessageStyleSuccess
	MessageStyleError
	MessageStyleInput
	MessageStyleOutput
)

//StyledText represent styled text
type StyledText struct {
	Text  string
	Style int
}

//NewStyledText creates a new message
func NewStyledText(text string, style int) *StyledText {
	return &StyledText{
		Text:  text,
		Style: style,
	}
}

//Message represent en event message, that is handled by CLI or Web reporter.
type Message struct {
	Header *StyledText
	Tag    *StyledText
	Items  []*StyledText
}

//NewMessage creates a new tag message
func NewMessage(header *StyledText, tag *StyledText, items ...*StyledText) *Message {
	return &Message{
		Header: header,
		Tag:    tag,
		Items:  items,
	}
}

//MessageReporter represents a reporter that can report tag messages
type MessageReporter interface {
	//Returns zero or more  messages
	Messages() []*Message
}

//RepeatedMessage represents a repeated message
type RepeatedMessage struct {
	Total int
	Count int
	Type  string
}

//RepeatedReporter represents a reporter that overrides current line (with \r)
type RepeatedReporter interface {
	//Returns messages
	Message(repeated *RepeatedMessage) *Message
}

//RunnerInput represent event storing runner input data, this interface enables matching runner in/out with failed validation (CLI)
type RunnerInput interface {
	IsInput() bool
}

//RunnerOutput represent event storing runner output data,this interface enables matching runner in/out with failed validation(CLI)
type RunnerOutput interface {
	IsOutput() bool
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

func (e *Event) Package() string {
	var eventType = fmt.Sprintf("%T", e.Value)
	var fragments = strings.Split(eventType, ".")
	return strings.Replace(fragments[0], "*", "", 1)
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

func (e *SleepEvent) Message(repeated *RepeatedMessage) *Message {
	var tag = NewStyledText("sleep", MessageStyleGeneric)
	var title *StyledText
	repeated.Total = + int(time.Millisecond * time.Duration(e.SleepTimeMs))
	if repeated.Count == 0 {
		title = NewStyledText(fmt.Sprintf("%v ms", e.SleepTimeMs), MessageStyleGeneric)
	} else {
		var sleptSoFar = time.Millisecond * time.Duration(repeated.Total)
		title = NewStyledText(fmt.Sprintf("%v ms x %v,  slept so far: %v", e.SleepTimeMs, repeated.Count, sleptSoFar), MessageStyleGeneric)
	}
	return NewMessage(title, tag)
}

//NewSleepEvent create a new sleep event
func NewSleepEvent(sleepTimeMs int) *SleepEvent {
	return &SleepEvent{SleepTimeMs: sleepTimeMs}
}

//ErrorEvent represents a Sleep
type ErrorEvent struct {
	Error string
}

//Messages returns messages
func (e *ErrorEvent) Messages() []*Message {
	return []*Message{
		NewMessage(NewStyledText(fmt.Sprintf("%v", e.Error), MessageStyleError), NewStyledText("error", MessageStyleError))}
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

//AsEventListener adds Listener
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
