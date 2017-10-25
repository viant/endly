package endly

import (
	"fmt"
)

//SdkServiceID represents system sdk
const SdkServiceID = "sdk"

type systemSdkService struct {
	*AbstractService
	jdkService *systemJdkService
}

func (s *systemSdkService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))

	var err error
	switch actualRequest := request.(type) {
	case *SystemSdkSetRequest:
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

func (s *systemSdkService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "set":
		return &SystemSdkSetRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)

}

func (s *systemSdkService) setSdk(context *Context, request *SystemSdkSetRequest) (*SystemSdkSetResponse, error) {
	switch request.Sdk {
	case "jdk":
		return s.jdkService.setSdk(context, request)
	}
	return nil, fmt.Errorf("Unsupported jdk: %v", request.Sdk)
}

//NewSystemJdkService creates a new system jdk service.
func NewSystemJdkService() Service {
	var result = &systemSdkService{
		jdkService:      &systemJdkService{},
		AbstractService: NewAbstractService(SdkServiceID),
	}
	result.AbstractService.Service = result
	return result
}
