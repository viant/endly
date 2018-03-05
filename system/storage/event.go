package storage

import (
	"fmt"
	"github.com/viant/endly"
)

var removeFilter = endly.NewFilteredReporter("storage", "storage.remove")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *RemoveRequest) CanReport(filter map[string]bool) bool {
	return removeFilter.CanReport(filter)
}

//Messages returns tag messages
func (r *RemoveRequest) Messages() []*endly.Message {
	if len(r.Resources) == 0 {
		return []*endly.Message{}
	}
	var fragments = make([]*endly.StyledText, 0)
	for _, resource := range r.Resources {
		fragments = append(fragments, endly.NewStyledText(fmt.Sprintf("SourceURL: %v", resource.URL), endly.MessageStyleInput))
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("remove", endly.MessageStyleGeneric),
		fragments...),
	}
}

var uploadFilter = endly.NewFilteredReporter("storage", "storage.upload")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *UploadRequest) CanReport(filter map[string]bool) bool {
	return uploadFilter.CanReport(filter)
}

//Messages returns event messages
func (r *UploadRequest) Messages() []*endly.Message {
	if r.Target == nil {
		return []*endly.Message{}
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("upload", endly.MessageStyleGeneric),
		endly.NewStyledText(fmt.Sprintf("SourcKey: %v", r.SourceKey), endly.MessageStyleInput),
		endly.NewStyledText(fmt.Sprintf("TargetURL: %v", r.Target.URL), endly.MessageStyleOutput),
	)}
}

var downloadFilter = endly.NewFilteredReporter("storage", "storage.download")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *DownloadRequest) CanReport(filter map[string]bool) bool {
	return downloadFilter.CanReport(filter)
}

//Messages returns event messages
func (r *DownloadRequest) Messages() []*endly.Message {
	if r.Source == nil {
		return []*endly.Message{}
	}
	return []*endly.Message{endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
		endly.NewStyledText("upload", endly.MessageStyleGeneric),
		endly.NewStyledText(fmt.Sprintf("Source: %v", r.Source.URL), endly.MessageStyleInput),
		endly.NewStyledText(fmt.Sprintf("TargetKey: %v", r.TargetKey), endly.MessageStyleOutput),
	)}
}

var copyFilter = endly.NewFilteredReporter("storage", "storage.copy")

//CanReport returns true if filter has matching event key enabled reporting option
func (r *CopyRequest) CanReport(filter map[string]bool) bool {
	return copyFilter.CanReport(filter)
}

//Messages returns event messages
func (r *CopyRequest) Messages() []*endly.Message {
	if len(r.Transfers) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for _, transfer := range r.Transfers {
		if transfer.Source == nil || transfer.Target == nil {
			continue
		}
		result = append(result, endly.NewMessage(endly.NewStyledText("", endly.MessageStyleGeneric),
			endly.NewStyledText("copy", endly.MessageStyleGeneric),
			endly.NewStyledText(fmt.Sprintf("SourceURL: %v", transfer.Source.URL), endly.MessageStyleInput),
			endly.NewStyledText(fmt.Sprintf("TargetURL: %v", transfer.Target.URL), endly.MessageStyleOutput),
		))
	}
	return result
}
