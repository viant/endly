package endly

import (
	"fmt"
)

const JsdServiceId = "jdkService"

type SetSdkResponse struct {
	Home  string
	Build string
}

type SetSdkRequest struct {
	Sdk     string
	Version string
	Target  *Resource
}

type sdkService struct {
	*AbstractService
	jdkService *jdkService
}

func (s *sdkService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{
		Status: "ok",
	}
	var err error
	switch actualRequest := request.(type) {
	case *SetSdkRequest:
		response.Response, err = s.setSdk(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run sdk: %v, %v", actualRequest.Sdk, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (t *sdkService) NewRequest(action string) (interface{}, error) {
	return &SetSdkRequest{}, nil
}

func (s *sdkService) setSdk(context *Context, request *SetSdkRequest) (*SetSdkResponse, error) {
	switch request.Sdk {
	case "jdk":
		return s.jdkService.setSdk(context, request)
	}
	return nil, fmt.Errorf("Unsupported jdk: %v\n", request.Sdk)
}

func NewJdkService() Service {
	var result = &sdkService{
		jdkService:      &jdkService{},
		AbstractService: NewAbstractService(JsdServiceId),
	}
	result.AbstractService.Service = result
	return result
}
