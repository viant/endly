package apigateway

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
)

type RestMethod struct {
	HTTPMethod        string
	URI               *string
	Type              *string
	AuthorizationType *string
}

type RestResource struct {
	Path    string
	ID      string
	Methods []*RestMethod
	TestCLI string
}

type ResetAPIEvent struct {
	Name      string
	ID        string
	Endpoint  string
	Resources []*RestResource
}

func (e *ResetAPIEvent) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(e.Name, msg.MessageStyleGeneric),
			msg.NewStyled("restAPI", msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
		),
	}
}

func NewResetAPIEvent(output *SetupRestAPIOutput) *ResetAPIEvent {
	result := &ResetAPIEvent{
		Name:      *output.Name,
		ID:        *output.Id,
		Endpoint:  output.EndpointURL,
		Resources: make([]*RestResource, 0),
	}
	if len(output.Resources) == 0 {
		return result
	}
	for _, resource := range output.Resources {
		restResource := &RestResource{
			Path:    *resource.Path,
			ID:      *resource.Id,
			Methods: make([]*RestMethod, 0),
			TestCLI:fmt.Sprintf(`aws apigateway test-invoke-method --rest-api-id %s  --resource-id %s --http-method "GET"`, *output.Id, *resource.Id),
		}
		if len(resource.ResourceMethods) > 0 {
			for k, v := range resource.ResourceMethods {
				method := &RestMethod{
					HTTPMethod:        k,
					AuthorizationType: v.AuthorizationType,
				}
				if v.MethodIntegration != nil {
					method.URI = v.MethodIntegration.Uri
					method.Type = v.MethodIntegration.Type
				}
				restResource.Methods = append(restResource.Methods, method)
			}
		}
		result.Resources = append(result.Resources, restResource)
	}
	return result
}

//Messages returns messages
func (o *SetupRestAPIOutput) Messages() []*msg.Message {
	if o == nil {
		return nil
	}
	event := NewResetAPIEvent(o)
	return event.Messages()
}
