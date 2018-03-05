package sdk

import (
	"fmt"
	"github.com/viant/endly"
)

var sdkFilter = endly.NewFilteredReporter("sdk")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *SetRequest) CanReport(filter map[string]bool) bool {
	return sdkFilter.CanReport(filter)
}

//Messages returns tag messages
func (r *SetRequest) Messages() []*endly.Message {
	var header = fmt.Sprintf("%v %v", r.Sdk, r.Version)
	return []*endly.Message{endly.NewMessage(endly.NewStyledText(header, endly.MessageStyleGeneric),
		endly.NewStyledText("sdk", endly.MessageStyleGeneric))}
}
