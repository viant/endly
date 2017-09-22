package endly

import (
	"fmt"
)

const SdkServiceId = "sdk"

type SdkSetResponse struct {
	Home  string
	Build string
}

type SdkSetRequest struct {
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
	case *SdkSetRequest:
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
	return &SdkSetRequest{}, nil
}

func (s *sdkService) setSdk(context *Context, request *SdkSetRequest) (*SdkSetResponse, error) {
	switch request.Sdk {
	case "jdk":
		return s.jdkService.setSdk(context, request)
	}
	return nil, fmt.Errorf("Unsupported jdk: %v\n", request.Sdk)
}

func NewJdkService() Service {
	var result = &sdkService{
		jdkService:      &jdkService{},
		AbstractService: NewAbstractService(SdkServiceId),
	}
	result.AbstractService.Service = result
	return result
}
