package selenium

import (
	"github.com/viant/endly"
)

//Messages returns messages
func (r *RunResponse) Messages() []*endly.Message {
	var result = make([]*endly.Message, 0)
	if len(r.LookupErrors) == 0 {
		return result
	}
	for _, errMessage := range r.LookupErrors {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(errMessage, endly.MessageStyleError), endly.NewStyledText("selenium", endly.MessageStyleGeneric)))
	}
	return result
}
