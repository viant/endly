package dsunit

import (
	"fmt"
	"github.com/viant/endly/model/msg"
)

//Messages returns messages
func (r *InitRequest) Messages() []*msg.Message {
	if r.RegisterRequest == nil {
		return []*msg.Message{}
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
func (r *RegisterRequest) Messages() []*msg.Message {
	if r.Config == nil {
		return []*msg.Message{}
	}
	var descriptor = r.Config.SecureDescriptor
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(fmt.Sprintf("Datastore: %v, %v:%v", r.Datastore, r.Config.DriverName, descriptor), msg.MessageStyleGeneric), msg.NewStyled("register", msg.MessageStyleGeneric)),
	}
}

//Messages returns messages
func (r *MappingRequest) Messages() []*msg.Message {
	if len(r.Mappings) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for _, mapping := range r.Mappings {
		result = append(result,
			msg.NewMessage(msg.NewStyled(fmt.Sprintf("(%v) %v", mapping.Name, mapping.URL), msg.MessageStyleGeneric), msg.NewStyled("mapping", msg.MessageStyleGeneric)))
	}
	return result

}

//Messages returns messages
func (r *RunScriptRequest) Messages() []*msg.Message {
	if len(r.Scripts) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for _, script := range r.Scripts {
		result = append(result,
			msg.NewMessage(msg.NewStyled(fmt.Sprintf("(%v) %v", r.Datastore, script.URL), msg.MessageStyleGeneric), msg.NewStyled("sql", msg.MessageStyleGeneric)))

	}
	return result
}

//Messages returns messages
func (r *SequenceResponse) Messages() []*msg.Message {
	if len(r.Sequences) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for table, seq := range r.Sequences {
		result = append(result,
			msg.NewMessage(msg.NewStyled(fmt.Sprintf("%30s: %v", table, seq), msg.MessageStyleGeneric), msg.NewStyled("seq", msg.MessageStyleGeneric)))
	}
	return result
}

//Messages returns messages
func (r *PrepareRequest) Messages() []*msg.Message {
	_ = r.Load()
	if r.DatasetResource == nil || len(r.Datasets) == 0 {
		return []*msg.Message{}
	}
	var result = make([]*msg.Message, 0)
	for _, dataset := range r.Datasets {
		result = append(result,
			msg.NewMessage(msg.NewStyled(fmt.Sprintf("(%v) %v: %v", r.Datastore, dataset.Table, len(dataset.Records)), msg.MessageStyleGeneric), msg.NewStyled("populate", msg.MessageStyleGeneric)))
	}
	return result
}

//Messages returns messages
func (r *QueryRequest) Messages() []*msg.Message {
	message := msg.NewMessage(msg.NewStyled(fmt.Sprintf("(%v) %v", r.Datastore, r.SQL), msg.MessageStyleGeneric), msg.NewStyled("populate", msg.MessageStyleGeneric))
	return []*msg.Message{message}
}
