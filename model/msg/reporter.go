package msg

// MessagesEvent transports already-materialized reporter messages.
type MessagesEvent struct {
	Reported []*Message
}

func (e *MessagesEvent) Messages() []*Message {
	return e.Reported
}
