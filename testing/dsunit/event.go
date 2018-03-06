package dsunit

import (
	"fmt"
	"github.com/viant/endly"
)

//Messages returns messages
func (r *InitRequest) Messages() []*endly.Message {
	if r.RegisterRequest == nil {
		return []*endly.Message{}
	}
	var registerRequest = RegisterRequest(*r.RegisterRequest)
	var result = registerRequest.Messages()
	if r.RunScriptRequest != nil {
		var scriptRequest = RunScriptRequest(*r.RunScriptRequest)
		result = append(result, scriptRequest.Messages()...)
	}
	if r.MappingRequest != nil {
		var mappings = MappingRequest(*r.MappingRequest)
		result = append(result, mappings.Messages()...)
	}
	return result
}

//Messages returns messages
func (r *RegisterRequest) Messages() []*endly.Message {
	if r.Config == nil {
		return []*endly.Message{}
	}
	var descriptor = r.Config.SecureDescriptor
	return []*endly.Message{
		endly.NewMessage(endly.NewStyledText(fmt.Sprintf("Datastore: %v, %v:%v", r.Datastore, r.Config.DriverName, descriptor), endly.MessageStyleGeneric), endly.NewStyledText("register", endly.MessageStyleGeneric)),
	}
}

//Messages returns messages
func (r *MappingRequest) Messages() []*endly.Message {
	if len(r.Mappings) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for _, mapping := range r.Mappings {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(fmt.Sprintf("(%v) %v", mapping.Name, mapping.URL), endly.MessageStyleGeneric), endly.NewStyledText("mapping", endly.MessageStyleGeneric)))
	}
	return result

}

//Messages returns messages
func (r *RunScriptRequest) Messages() []*endly.Message {
	if len(r.Scripts) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for _, script := range r.Scripts {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(fmt.Sprintf("(%v) %v", r.Datastore, script.URL), endly.MessageStyleGeneric), endly.NewStyledText("sql", endly.MessageStyleGeneric)))

	}
	return result
}

//Messages returns messages
func (r *SequenceResponse) Messages() []*endly.Message {
	if len(r.Sequences) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for table, seq := range r.Sequences {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(fmt.Sprintf("%30s: %v", table, seq), endly.MessageStyleGeneric), endly.NewStyledText("seq", endly.MessageStyleGeneric)))
	}
	return result
}

//Messages returns messages
func (r *PrepareRequest) Messages() []*endly.Message {
	r.Load()
	if r.DatasetResource == nil || len(r.Datasets) == 0 {
		return []*endly.Message{}
	}
	var result = make([]*endly.Message, 0)
	for _, dataset := range r.Datasets {
		result = append(result,
			endly.NewMessage(endly.NewStyledText(fmt.Sprintf("(%v) %v: %v", r.Datastore, dataset.Table, len(dataset.Records)), endly.MessageStyleGeneric), endly.NewStyledText("populate", endly.MessageStyleGeneric)))
	}
	return result
}
