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
	Header     *StyledText
	Tag        *StyledText
	Messages   []*StyledText
	Counter    int
}

//NewMessage creates a new tag message
func NewMessage(header *StyledText, tag *StyledText,  messages ...*StyledText) *Message {
	return &Message{
		Header:     header,
		Tag:        tag,
		Messages:   messages,
	}
}

//FilteredReporter represents reporter that can filter messages
type FilteredReporter interface {
	//Returns true if can report for supplied filter
	CanReport(filter map[string]bool) bool
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
}

//RepeatedReporter represents a reporter that updted current line (with \r)
type RepeatedReporter interface {
	//Returns messages
	Message(repeated *RepeatedMessage) *Message
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

func (e *SleepEvent) Message(repeated *RepeatedMessage) *Message {

	var result *Message
	var tagText = NewStyledText("sleep", MessageStyleGeneric)
	var title *StyledText

	if repeated != nil {
		counter += repeated.Total
	}

	var sleepTime = time.Millisecond *  time.Duration(counter)
	if  repeated.Count == 0 {
		title = fmt.Sprintf("%v ms", e.SleepTimeMs)
	} else {
		var timeSoFar
		title = fmt.Sprintf("%v ms x %v,  slept so far: %v", e.SleepTimeMs, repeated.Count, time.Millisecond * repeated.Total)

	}
	result = NewMessage(NewStyledText(fmt.Sprintf("%v", e.SessionID), MessageStyleGeneric), NewStyledText("stdout", endly.MessageStyleGeneric),



		result = NewMessage(NewStyledText(fmt.Sprintf("%v", e.SessionID), MessageStyleGeneric), NewStyledText("stdout", endly.MessageStyleGeneric),

		r.overrideShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms x %v,  slept so far: %v", actual.SleepTimeMs, r.SleepCount, r.SleepTime), endly.MessageStyleGeneric, "Sleep")
	} else {
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v", e.SessionID), endly.MessageStyleGeneric), endly.NewStyledText("stdout", endly.MessageStyleGeneric),
		r.SleepTime = 0
		r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms", actual.SleepTimeMs), endly.MessageStyleGeneric, "Sleep")
	}
}


/*

	case *endly.SleepEvent:
		if r.SleepCount > 0 {
			r.overrideShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms x %v,  slept so far: %v", actual.SleepTimeMs, r.SleepCount, r.SleepTime), endly.MessageStyleGeneric, "Sleep")
		} else {
			r.SleepTime = 0
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms", actual.SleepTimeMs), endly.MessageStyleGeneric, "Sleep")
		}

		r.SleepTagID = r.eventTag.TagID
		r.SleepTime += time.Millisecond * time.Duration(actual.SleepTimeMs)
		r.SleepCount++
		return true
	}

*/


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

//NewFilteredReporter creates a new abstract reporter
func NewFilteredReporter(serviceKey string, eventKeys ...string) FilteredReporter {
	var result = &AbstractReporter{
		serviceKey: strings.ToLower(serviceKey),
		eventKeys:  eventKeys,
	}
	if len(result.eventKeys) == 0 {
		result.eventKeys = []string{}
	}
	for i, eventKey := range result.eventKeys {
		result.eventKeys[i] = strings.ToLower(eventKey)
	}
	return result
}
