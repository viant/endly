package msg

//RepeatedEvent represents a generic repeated message
type RepeatedEvent struct {
	tag     string
	message string
}

func (e *RepeatedEvent) Message(repeated *Repeated) *Message {
	var tag = NewStyled(e.tag, MessageStyleGeneric)
	var title = NewStyled(e.message, MessageStyleGeneric)
	return NewMessage(title, tag)
}

//NewSleepEvent create a new sleep event
func NewRepeatedEvent(message, tag string) *RepeatedEvent {
	return &RepeatedEvent{message: message, tag: tag}
}
