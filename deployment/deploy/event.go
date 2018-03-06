package deploy

import (
	"fmt"
	"github.com/viant/endly"
)

//Items returns tag messages
func (r *Request) Messages() []*endly.Message {
	var header = fmt.Sprintf("app: %v:%v, forced: %v", r.AppName, r.Version, r.Force)
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(header, endly.MessageStyleGeneric),
		endly.NewStyledText("deploy", endly.MessageStyleGeneric))}
}
