package smtp

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
	"sync"
)

// Messages represents a FIFO message collection grouped  by user
type Messages struct {
	*sync.Mutex
	byUser  map[string][]*Message
	debug   bool
	context *endly.Context
}

// Push appends a message by user
func (m *Messages) Push(username string, message *Message) {
	m.Lock()
	defer m.Unlock()
	if m.debug {
		info, _ := toolbox.AsJSONText(message)
		_ = endly.Run(m.context, &workflow.PrintRequest{
			Style:   msg.MessageStyleOutput,
			Message: fmt.Sprintf("push [%v] <- %v", username, info),
		}, nil)
	}
	_, ok := m.byUser[username]
	if !ok {
		m.byUser[username] = make([]*Message, 0)
	}
	m.byUser[username] = append(m.byUser[username], message)
}

// Shift remove first placed message for supplied message
func (m *Messages) Shift(username string) *Message {
	m.Lock()
	defer m.Unlock()
	messages, ok := m.byUser[username]
	if !ok {
		return nil
	}
	if len(messages) == 0 {
		return nil
	}
	message := messages[0]
	m.byUser[username] = messages[1:]
	if m.debug {
		info, _ := toolbox.AsJSONText(message)
		_ = endly.Run(m.context, &workflow.PrintRequest{
			Style:   msg.MessageStyleOutput,
			Message: fmt.Sprintf("shift [%v] -> %v", username, info),
		}, nil)
	}
	return message
}

// NewMessages returns a new FIFO message collection by user
func NewMessages() *Messages {
	return &Messages{
		byUser: make(map[string][]*Message),
		Mutex:  &sync.Mutex{},
	}
}
