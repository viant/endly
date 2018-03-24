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
	Message(repeated *Repeated) *Message
}

//RunnerInput represent event storing runner input data, this interface enables matching runner in/out with failed validation (CLI)
type RunnerInput interface {
	IsInput() bool
}

//RunnerOutput represent event storing runner output data,this interface enables matching runner in/out with failed validation(CLI)
type RunnerOutput interface {
	IsOutput() bool
}

//Styled represent styled text
type Styled struct {
	Text  string
	Style int
}

//NewStyled creates a new message
func NewStyled(text string, style int) *Styled {
	return &Styled{
		Text:  text,
		Style: style,
	}
}

//Message represent en event message, that is handled by CLI or Web reporter.
type Message struct {
	Header *Styled
	Tag    *Styled
	Items  []*Styled
}

//NewMessage creates a new tag message
func NewMessage(header *Styled, tag *Styled, items ...*Styled) *Message {
	return &Message{
		Header: header,
		Tag:    tag,
		Items:  items,
	}
}

//Reporter represents a reporter that can report tag messages
type Reporter interface {
	//Returns zero or more  messages
	Messages() []*Message
}

//Repeated represents a repeated message
type Repeated struct {
	Spent time.Duration
	Count int
	Type  string
}
