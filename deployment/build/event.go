package build

import (
	"fmt"
	"github.com/viant/endly"
)

//Items returns tag messages
func (r *Request) Messages() []*endly.Message {
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v %v", r.BuildSpec.Name, r.Target.URL), endly.MessageStyleGeneric),
		endly.NewStyledText("build", endly.MessageStyleGeneric))}
}
