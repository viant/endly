package apigateway

import (
	"fmt"
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
)

// Represents reset method event part
type RestMethodInfo struct {
	HTTPMethod        string
	URI               *string
	Type              *string
	AuthorizationType *string
	AuthorizerID      *string
}

// Represents reset resource event part
type RestResourceInfo struct {
	Path    string
	ID      string
	Methods []*RestMethodInfo
	TestCLI string
}

// ResetAPIEvent represents rest API event
type ResetAPIEvent struct {
	Name      string
	ID        string
	Endpoint  string
	Resources []*RestResourceInfo
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
		Resources: make([]*RestResourceInfo, 0),
	}
	if len(output.Resources) == 0 {
		return result
	}
	for _, resource := range output.Resources {
		restResource := &RestResourceInfo{
			Path:    *resource.Path,
			ID:      *resource.Id,
			Methods: make([]*RestMethodInfo, 0),
			TestCLI: fmt.Sprintf(`aws apigateway test-invoke-method --rest-api-id %s  --resource-id %s --http-method "GET"`, *output.Id, *resource.Id),
		}
		if len(resource.ResourceMethods) > 0 {

			for k, v := range resource.ResourceMethods {
				method := &RestMethodInfo{
					HTTPMethod:        k,
					AuthorizationType: v.AuthorizationType,
					AuthorizerID:      v.AuthorizerId,
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

// Messages returns messages
func (o *SetupRestAPIOutput) Messages() []*msg.Message {
	if o == nil {
		return nil
	}
	event := NewResetAPIEvent(o)
	return event.Messages()
}
