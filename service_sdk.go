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

func (s *sdkService) Run(context *Context, request interface{}) *Response {
	var response = &Response{
		Status: "ok",
	}
	switch castedRequest := request.(type) {
	case *SetSdkRequest:
		response.Response, response.Error = s.setSdk(context, castedRequest)
	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}
	if response.Error != nil {
		response.Status = "error"
	}
	return response
}

func (t *sdkService) NewRequest(name string) (interface{}, error) {
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

