package msg

import "sync"

//Events represents events
type Events struct {
	mux    *sync.Mutex
	Events []Event
}

//AsListener returns a listener
func (e *Events) AsListener() Listener {
	return func(event Event) {
		e.mux.Lock()
		defer e.mux.Unlock()
		e.Events = append(e.Events, event)
	}
}

//NewEvents creates a new events
func NewEvents() *Events {
	return &Events{
		mux:    &sync.Mutex{},
		Events: make([]Event, 0),
	}
}
