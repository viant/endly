package logs

import (
	"github.com/viant/toolbox"
)

func normalizeMessages(messages []interface{}) {
	for i, message := range messages {
		textMessagePrt, ok := message.(*string)
		if ! ok || textMessagePrt == nil {
			continue
		}
		textMessage := *textMessagePrt
		if toolbox.IsCompleteJSON(textMessage)  {
			if data, err := toolbox.JSONToInterface(textMessage);err == nil {
				messages[i] = data
			}
		}
	}
}