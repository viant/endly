package msg

import (
	"fmt"
	"time"
)

// SleepEvent represents a Sleep
type SleepEvent struct {
	SleepTimeMs int
}

func (e *SleepEvent) Message(repeated *Repeated) *Message {
	var tag = NewStyled("sleep", MessageStyleGeneric)
	var title *Styled
	duration := (time.Millisecond * time.Duration(e.SleepTimeMs))
	repeated.Spent += duration
	if repeated.Count == 0 {
		title = NewStyled(fmt.Sprintf("%v ms", e.SleepTimeMs), MessageStyleGeneric)
	} else {
		var sleptSoFar = repeated.Spent
		title = NewStyled(fmt.Sprintf("%v ms x %v,  slept so far: %v", e.SleepTimeMs, repeated.Count+1, sleptSoFar), MessageStyleGeneric)
	}
	return NewMessage(title, tag)
}

// NewSleepEvent create a new sleep event
func NewSleepEvent(sleepTimeMs int) *SleepEvent {
	return &SleepEvent{SleepTimeMs: sleepTimeMs}
}
