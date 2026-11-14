package cli

import (
	"time"

	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
)

// reportEventTime preserves the timestamp without retaining the source event.
type reportEventTime struct {
	msg.Event
	at time.Time
}

func (e reportEventTime) Timestamp() time.Time { return e.at }

func retainReportEvent(event msg.Event) msg.Event {
	init, ok := event.Value().(*workflow.InitEvent)
	if !ok || init == nil {
		return event
	}
	// Rendering and other listeners receive the complete event. The final CLI
	// report needs its boundary and timestamp, but never reads the workflow state.
	// Retaining that copy here otherwise keeps every hydrated fixture snapshot
	// alive for the entire suite. Request, response and assertion events are kept.
	retained := msg.NewEventWithInit(&workflow.InitEvent{Tasks: init.Tasks}, event.Init())
	retained.SetLoggable(event.IsLoggable())
	return reportEventTime{Event: retained, at: event.Timestamp()}
}
