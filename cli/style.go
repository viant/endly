package cli

import "github.com/viant/endly/model/msg"

type Style struct {
	MessageStyleColor  map[int]string
	InputColor         string
	OutputColor        string
	TagColor           string
	InverseTag         bool
	ServiceActionColor string
	PathColor          string
	SuccessColor       string
	ErrorColor         string
}

func NewStyle() *Style {
	return &Style{
		InputColor:         "blue",
		OutputColor:        "green",
		TagColor:           "brown",
		PathColor:          "brown",
		ServiceActionColor: "gray",
		ErrorColor:         "red",
		InverseTag:         true,
		MessageStyleColor: map[int]string{
			messageTypeTagDescription: "cyan",
			msg.MessageStyleError:     "red",
			msg.MessageStyleSuccess:   "green",
			msg.MessageStyleGeneric:   "black",
			msg.MessageStyleInput:     "blue",
			msg.MessageStyleOutput:    "green",
			msg.MessageStyleGroup:     "bold",
		},
	}
}
