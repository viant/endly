package msg

import (
	"time"
	"fmt"
)


//SleepEvent represents a Sleep
type SleepEvent struct {
	SleepTimeMs int
}

func (e *SleepEvent) Message(repeated *RepeatedMessage) *Message {
	var tag = NewStyledText("sleep", MessageStyleGeneric)
	var title *StyledText
	repeated.Spent += (time.Millisecond * time.Duration(e.SleepTimeMs))
	if repeated.Count == 0 {
		title = NewStyledText(fmt.Sprintf("%v ms", e.SleepTimeMs), MessageStyleGeneric)
	} else {
		var sleptSoFar = repeated.Spent
		title = NewStyledText(fmt.Sprintf("%v ms x %v,  slept so far: %v", e.SleepTimeMs, repeated.Count+1, sleptSoFar), MessageStyleGeneric)
	}
	return NewMessage(title, tag)
}

//NewSleepEvent create a new sleep event
func NewSleepEvent(sleepTimeMs int) *SleepEvent {
	return &SleepEvent{SleepTimeMs: sleepTimeMs}
}
