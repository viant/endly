package storage

import (
	"fmt"
	"github.com/viant/endly"
)

//Items returns tag messages
func (r *RemoveRequest) Messages() []*endly.Message {
	if len(r.Assets) == 0 {
		return []*endly.Message{}
	}
	var fragments = make([]*endly.StyledText, 0)
	for _, resource := range r.Assets {
		fragments = append(fragments, endly.NewStyledText(fmt.Sprintf("SourceURL: %v", resource.URL), endly.MessageStyleInput))
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("remove", endly.MessageStyleGeneric),
		fragments...),
	}
}

//Items returns event messages
func (r *UploadRequest) Messages() []*endly.Message {
	if r.Dest == nil {
		return []*endly.Message{}
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("upload", endly.MessageStyleGeneric),
		endly.NewStyledText(fmt.Sprintf("SourcKey: %v", r.SourceKey), endly.MessageStyleInput),
		endly.NewStyledText(fmt.Sprintf("DestURL: %v", r.Dest.URL), endly.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *DownloadRequest) Messages() []*endly.Message {
	if r.Source == nil {
		return []*endly.Message{}
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("upload", endly.MessageStyleGeneric),
		endly.NewStyledText(fmt.Sprintf("Source: %v", r.Source.URL), endly.MessageStyleInput),
		endly.NewStyledText(fmt.Sprintf("DestKey: %v", r.DestKey), endly.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *CopyRequest) Messages() []*endly.Message {
	r.Init()
	if len(r.Transfers) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for _, transfer := range r.Transfers {
		if transfer.Source == nil || transfer.Dest == nil {
			continue
		}
		result = append(result, endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
			endly.NewStyledText("copy", endly.MessageStyleGeneric),
			endly.NewStyledText(fmt.Sprintf("SourceURL: %v", transfer.Source.URL), endly.MessageStyleInput),
			endly.NewStyledText(fmt.Sprintf("DestURL: %v", transfer.Dest.URL), endly.MessageStyleOutput),
		))
	}
	return result
}
