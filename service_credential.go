package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

const CredentialServiceId = "credential"


type CredentailSetRequest struct {
	Aliases map[string]string
}

type credentialService struct {
	*AbstractService
}


func (s *credentialService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *CredentailSetRequest:
		response.Response, err = s.set(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run script: %v, %v", actualRequest.Aliases, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}
func (s *credentialService) set(context *Context, request *CredentailSetRequest) (interface{}, error) {
	manager, err := context.Manager()
	if err != nil {
		return nil, err
	}
	for name, file := range request.Aliases {
		if ! toolbox.FileExists(file) {
			return nil, fmt.Errorf("Failed to register %v - credentail file %v does not exists", name, file)
		}
		manager.RegisterCredentialFile(name, file)
	}
	return nil, nil
}

func (s *credentialService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "set":
		return &CredentailSetRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewCredentialService() Service {
	var result = &credentialService{
		AbstractService: NewAbstractService(CredentialServiceId),
	}
	result.AbstractService.Service = result
	return result
}
