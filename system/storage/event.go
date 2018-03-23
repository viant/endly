package storage

import (
	"fmt"
	"github.com/viant/endly/msg"
)

//Items returns tag messages
func (r *RemoveRequest) Messages() []*msg.Message {
	if len(r.Assets) == 0 {
		return []*msg.Message{}
	}
	var fragments = make([]*msg.StyledText, 0)
	for _, resource := range r.Assets {
		fragments = append(fragments, msg.NewStyledText(fmt.Sprintf("SourceURL: %v", resource.URL), msg.MessageStyleInput))
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyledText("", msg.MessageStyleGeneric),
		msg.NewStyledText("remove", msg.MessageStyleGeneric),
		fragments...),
	}
}

//Items returns event messages
func (r *UploadRequest) Messages() []*msg.Message {
	if r.Dest == nil {
		return []*msg.Message{}
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyledText("", msg.MessageStyleGeneric),
		msg.NewStyledText("upload", msg.MessageStyleGeneric),
		msg.NewStyledText(fmt.Sprintf("SourcKey: %v", r.SourceKey), msg.MessageStyleInput),
		msg.NewStyledText(fmt.Sprintf("DestURL: %v", r.Dest.URL), msg.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *DownloadRequest) Messages() []*msg.Message {
	if r.Source == nil {
		return []*msg.Message{}
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyledText("", msg.MessageStyleGeneric),
		msg.NewStyledText("upload", msg.MessageStyleGeneric),
		msg.NewStyledText(fmt.Sprintf("Source: %v", r.Source.URL), msg.MessageStyleInput),
		msg.NewStyledText(fmt.Sprintf("DestKey: %v", r.DestKey), msg.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *CopyRequest) Messages() []*msg.Message {
	r.Init()
	if len(r.Transfers) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for _, transfer := range r.Transfers {
		if transfer.Source == nil || transfer.Dest == nil {
			continue
		}
		result = append(result, msg.NewMessage(msg.NewStyledText("", msg.MessageStyleGeneric),
			msg.NewStyledText("copy", msg.MessageStyleGeneric),
			msg.NewStyledText(fmt.Sprintf("SourceURL: %v", transfer.Source.URL), msg.MessageStyleInput),
			msg.NewStyledText(fmt.Sprintf("DestURL: %v", transfer.Dest.URL), msg.MessageStyleOutput),
		))
	}
	return result
}
