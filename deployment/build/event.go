package build

import (
	"fmt"
	"github.com/viant/endly/msg"
)

//Items returns tag messages
func (r *Request) Messages() []*msg.Message {
	return []*msg.Message{msg.NewMessage(msg.NewStyledText(fmt.Sprintf("%v %v", r.BuildSpec.Name, r.Target.URL), msg.MessageStyleGeneric),
		msg.NewStyledText("build", msg.MessageStyleGeneric))}
}
