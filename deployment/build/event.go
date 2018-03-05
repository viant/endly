package build

import (
	"fmt"
	"github.com/viant/endly"
)

var buildFilter = endly.NewFilteredReporter("build")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *Request) CanReport(filter map[string]bool) bool {
	return buildFilter.CanReport(filter)
}

//Messages returns tag messages
func (r *Request) Messages() []*endly.Message {
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%v %v", r.BuildSpec.Name, r.Target.URL), endly.MessageStyleGeneric),
		endly.NewStyledText("build", endly.MessageStyleGeneric))}
}
