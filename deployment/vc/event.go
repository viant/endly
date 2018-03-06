package vc

import (
	"fmt"
	"github.com/viant/endly"
)

//Items returns tag messages
func (r *CheckoutRequest) Messages() []*endly.Message {
	if r.Origin == nil || r.Target == nil {
		return []*endly.Message{}
	}
	var header = fmt.Sprintf("%v %v", r.Origin.URL, r.Target.URL)
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(header, endly.MessageStyleGeneric),
		endly.NewStyledText("checkout", endly.MessageStyleGeneric))}
}
