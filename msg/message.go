package msg

import "time"

const (
	MessageStyleGeneric = iota
	MessageStyleSuccess
	MessageStyleError
	MessageStyleInput
	MessageStyleOutput
	MessageStyleGroup
)

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
	Spent time.Duration
	Count int
	Type  string
}

