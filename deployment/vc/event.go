package vc

import (
	"fmt"
	"github.com/viant/endly/model/msg"
)

// Items returns tag messages
func (r *CheckoutRequest) Messages() []*msg.Message {
	if r.Origin == nil || r.Dest == nil {
		return []*msg.Message{}
	}
	var header = fmt.Sprintf("%v %v", r.Origin.URL, r.Dest.URL)
	return []*msg.Message{msg.NewMessage(msg.NewStyled(header, msg.MessageStyleGeneric),
		msg.NewStyled("checkout", msg.MessageStyleGeneric))}
}
