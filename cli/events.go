package cli

import (
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
	"sync"
)

// Event represents an tagged event
type Event struct {
	Description string
	Caller      string
	TagID       string
	Index       string
	Events      []msg.Event
	Validation  []*assertly.Validation
	PassedCount int
	FailedCount int
	subEvent    *Event
}

// AddEvent add provided event
func (e *Event) AddEvent(event msg.Event) {
	if len(e.Events) == 0 {
		e.Events = make([]msg.Event, 0)
	}
	e.Events = append(e.Events, event)
}

// Events represents tags
type Events struct {
	*model.Activities
	activity   *model.Activity
	tags       []*Event
	indexedTag map[string]*Event
	eventTag   *Event
	mutex      *sync.RWMutex
}

// AddTag adds reporting tag
func (r *Events) AddTag(event *Event) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tags = append(r.tags, event)
	r.indexedTag[event.TagID] = event
}

// Event returns an event tag
func (r *Events) EventTag() *Event {
	if r.Len() == 0 {
		if r.eventTag == nil {
			r.eventTag = &Event{}
			r.tags = append(r.tags, r.eventTag)
			return r.eventTag
		}
	}
	activity := r.Last()

	r.mutex.RLock()
	_, has := r.indexedTag[activity.TagID]
	r.mutex.RUnlock()

	if !has {
		eventTag := &Event{
			Caller: activity.Caller,
			TagID:  activity.TagID,
			Index:  activity.TagIndex,
		}
		r.AddTag(eventTag)
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.indexedTag[activity.TagID]
}

// Push Events
func (r *Events) Push(activity *model.Activity) {
	r.Activities.Push(activity)
	if activity.TagIndex != "" {
		r.mutex.Lock()
		r.mutex.Unlock()
	}
	r.activity = activity
}

func (r *Events) TemplateEvent(context *endly.Context, candidateTagID string) *Event {
	tagID := candidateTagID
	if candidateTagID == "" {
		process := workflow.Last(context)
		if process != nil {
			activity := process.Last()
			if activity != nil {
				tagID = activity.TagID
			}
		}
	}
	event, ok := r.indexedTag[tagID]
	if !ok {
		event = &Event{TagID: tagID}
		r.AddTag(event)
	}

	if event.Index == "" {
		var upstreamEvent *Event
		r.Range(func(activity *model.Activity) bool {
			if activity.TagIndex != "" {
				if upstreamEvent, ok = r.indexedTag[activity.TagID]; !ok {
					upstreamEvent = &Event{TagID: activity.TagID}
					r.AddTag(upstreamEvent)
					return false
				}
			}
			return true
		}, true)
		if upstreamEvent != nil {
			upstreamEvent.subEvent = event
			return upstreamEvent
		}
	}

	return event
}

// NewEventTags returns new events
func NewEventTags() *Events {
	return &Events{
		Activities: model.NewActivities(),
		tags:       make([]*Event, 0),
		indexedTag: make(map[string]*Event),
		mutex:      &sync.RWMutex{},
	}
}
