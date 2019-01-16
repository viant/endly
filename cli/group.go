package cli

import (
	"github.com/viant/endly/model/msg"
	"time"
)

type MessageGroup struct {
	message     *msg.Message
	startTime   *time.Time
	item        *msg.Styled
	enabled     bool
	pendingLine bool
}

//EnableIfMatched enable group if matched or first message, returns true if previous group matched
func (g *MessageGroup) EnableIfMatched(message *msg.Message) bool {
	if message.Header == nil {
		return false
	}

	if hasPrevious := g.message != nil; !hasPrevious {
		g.Set(message)
		return false
	}

	if g.message.Header.Equals(message.Header) && g.message.Tag.Equals(message.Tag) {
		g.Set(message)
		return true
	}
	g.Reset()
	return false
}

func (g *MessageGroup) Set(message *msg.Message) {
	if g.startTime == nil {
		now := time.Now()
		g.startTime = &now
	}
	g.message = message
	g.enabled = true
}

func (g *MessageGroup) Reset() {
	g.message = nil
	g.enabled = false
	g.startTime = nil
	g.item = nil
}
