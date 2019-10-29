package storage

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"strings"
)

/*
Message events control runner reporter and stdout
*/

//Items returns tag messages
func (r *RemoveRequest) Messages() []*msg.Message {
	if len(r.Assets) == 0 {
		return []*msg.Message{}
	}
	var fragments = make([]*msg.Styled, 0)
	for _, resource := range r.Assets {
		fragments = append(fragments, msg.NewStyled(fmt.Sprintf("SourceURL: %v", resource.URL), msg.MessageStyleInput))
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("Remove", msg.MessageStyleGeneric),
		fragments...),
	}
}

//Items returns event messages
func (r *UploadRequest) Messages() []*msg.Message {
	if r.Dest == nil {
		return []*msg.Message{}
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("Upload", msg.MessageStyleGeneric),
		msg.NewStyled(fmt.Sprintf("SourcKey: %v", r.SourceKey), msg.MessageStyleInput),
		msg.NewStyled(fmt.Sprintf("DestURL: %v", r.Dest.URL), msg.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *DownloadRequest) Messages() []*msg.Message {
	if r.Source == nil {
		return []*msg.Message{}
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("Upload", msg.MessageStyleGeneric),
		msg.NewStyled(fmt.Sprintf("Source: %v", r.Source.URL), msg.MessageStyleInput),
		msg.NewStyled(fmt.Sprintf("DestKey: %v", r.DestKey), msg.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *CopyRequest) Messages() []*msg.Message {
	_ = r.Init()
	if len(r.Transfers) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for _, transfer := range r.Transfers {
		if transfer.Source == nil || transfer.Dest == nil {
			continue
		}
		result = append(result, msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
			msg.NewStyled("Copy", msg.MessageStyleGeneric),
			msg.NewStyled(fmt.Sprintf("compress: %v, expand: %v", transfer.Compress, transfer.Expand), msg.MessageStyleGeneric),
			msg.NewStyled(fmt.Sprintf("SourceURL: %v", transfer.Source.URL), msg.MessageStyleInput),
			msg.NewStyled(fmt.Sprintf("DestURL: %v", transfer.Dest.URL), msg.MessageStyleOutput),
		))
	}
	return result
}


//Items returns event messages
func (r *GenerateRequest) Messages() []*msg.Message {
	_ = r.Init()

	if r.Dest == nil {
		return []*msg.Message{}
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("Generate", msg.MessageStyleGeneric),
		msg.NewStyled(fmt.Sprintf("Size: %v", r.Size), msg.MessageStyleInput),
		msg.NewStyled(fmt.Sprintf("DestURL: %v", r.Dest.URL), msg.MessageStyleOutput),
	)}
}


//Items returns event messages
func (r *ListResponse) Messages() []*msg.Message {
	if r.Assets == nil {
		return []*msg.Message{}
	}
	assets := make([]string, 0)
	for i := range r.Assets {
		content := ""
		if len(r.Assets[i].Data) > 0 {
			content = fmt.Sprintf("\n%v\n\n", string(r.Assets[i].Data))
		}
		assets = append(assets, fmt.Sprintf("%s %v%v", r.Assets[i].Mode.String(), r.Assets[i].Name, content))
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("List", msg.MessageStyleGeneric),
		msg.NewStyled(fmt.Sprintf("Source: %v", r.URL), msg.MessageStyleInput),
		msg.NewStyled(strings.Join(assets, "\n")+"\n", msg.MessageStyleOutput),
	)}
}

//Items returns event messages
func (r *ExistsResponse) Messages() []*msg.Message {
	if r.Exists == nil {
		return []*msg.Message{}
	}
	assets := make([]string, 0)
	for URL, exists := range r.Exists {
		assets = append(assets, fmt.Sprintf("%s: %v", URL, exists))
	}
	return []*msg.Message{msg.NewMessage(msg.NewStyled("", msg.MessageStyleGeneric),
		msg.NewStyled("Exists", msg.MessageStyleGeneric),
		msg.NewStyled(strings.Join(assets, "\n")+"\n", msg.MessageStyleOutput),
	)}
}
