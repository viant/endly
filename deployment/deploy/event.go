package deploy

import (
	"fmt"
	"github.com/viant/endly/msg"
)

//Items returns tag messages
func (r *Request) Messages() []*msg.Message {
	var header = fmt.Sprintf("app: %v:%v, forced: %v", r.AppName, r.Version, r.Force)
	return []*msg.Message{msg.NewMessage(msg.NewStyled(header, msg.MessageStyleGeneric),
		msg.NewStyled("deploy", msg.MessageStyleGeneric))}
}
