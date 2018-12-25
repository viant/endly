package sdk

import (
	"fmt"
	"github.com/viant/endly/model/msg"
)

//Items returns tag messages
func (r *SetRequest) Messages() []*msg.Message {
	var header = fmt.Sprintf("%v %v", r.Sdk, r.Version)
	return []*msg.Message{msg.NewMessage(msg.NewStyled(header, msg.MessageStyleGeneric),
		msg.NewStyled("sdk", msg.MessageStyleGeneric))}
}
