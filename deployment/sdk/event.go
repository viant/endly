package sdk

import (
	"fmt"
	"github.com/viant/endly"
)

//Items returns tag messages
func (r *SetRequest) Messages() []*endly.Message {
	var header = fmt.Sprintf("%v %v", r.Sdk, r.Version)
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(header, endly.MessageStyleGeneric),
		endly.NewStyledText("sdk", endly.MessageStyleGeneric))}
}
