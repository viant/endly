package deploy

import (
	"fmt"
	"github.com/viant/endly"
)

var deployFilter = endly.NewFilteredReporter("deploy")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *Request) CanReport(filter map[string]bool) bool {
	return deployFilter.CanReport(filter)
}

//Messages returns tag messages
func (r *Request) Messages() []*endly.Message {
	var header = fmt.Sprintf("app: %v:%v, forced: %v", r.AppName, r.Version, r.Force)
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(header, endly.MessageStyleGeneric),
		endly.NewStyledText("deploy", endly.MessageStyleGeneric))}
}
