package slack

import "sync"

var messagesKey = "messages"

//Messages messages holder
type Messages struct {
	messages []*Message
	mux      *sync.Mutex
}

//Push append a message
func (m *Messages) Push(message *Message) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.messages = append(m.messages, message)
}

//Shift removes a message
func (m *Messages) Shift() *Message {
	m.mux.Lock()
	defer m.mux.Unlock()
	if len(m.messages) == 0 {
		return nil
	}
	result := m.messages[0]
	m.messages = m.messages[1:]
	return result
}

//NewMessages creates a new messages
func NewMessages() *Messages {
	return &Messages{
		messages: make([]*Message, 0),
		mux:      &sync.Mutex{},
	}
}
