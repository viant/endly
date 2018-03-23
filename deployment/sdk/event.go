package sdk

import (
	"fmt"
	"github.com/viant/endly/msg"
)

//Items returns tag messages
func (r *SetRequest) Messages() []*msg.Message {
	var header = fmt.Sprintf("%v %v", r.Sdk, r.Version)
	return []*msg.Message{msg.NewMessage(msg.NewStyledText(header, msg.MessageStyleGeneric),
		msg.NewStyledText("sdk", msg.MessageStyleGeneric))}
}
