package build

import (
	"fmt"
	"github.com/viant/endly/model/msg"
)

//Items returns tag messages
func (r *Request) Messages() []*msg.Message {
	return []*msg.Message{msg.NewMessage(msg.NewStyled(fmt.Sprintf("%v %v", r.BuildSpec.Name, r.Target.URL), msg.MessageStyleGeneric),
		msg.NewStyled("build", msg.MessageStyleGeneric))}
}
