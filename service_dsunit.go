package endly

import "fmt"

const DataStoreUnitServiceId = "dsunit"

type DsUnitPrepareRequest struct {
}

type DsUnitVerifyRequest struct {
}

type dsataStoreUnitService struct {
	*AbstractService
}

func (s *dsataStoreUnitService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *DsUnitPrepareRequest:
		response.Response, err = s.prepare(actualRequest)
		if err != nil {
			response.Response = fmt.Errorf("%v", err)
		}
	case *DsUnitVerifyRequest:
		response.Response, err = s.verify(actualRequest)
		if err != nil {
			response.Response = fmt.Errorf("%v", err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *dsataStoreUnitService) verify(request *DsUnitVerifyRequest) (interface{}, error) {

	return nil, nil
}

func (s *dsataStoreUnitService) prepare(request *DsUnitPrepareRequest) (interface{}, error) {

	return nil, nil
}

func (s *dsataStoreUnitService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "prepare":
		return &DsUnitPrepareRequest{}, nil
	case "verify":
		return &DsUnitVerifyRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}


func NewDataStoreUnitService() Service {
	var result = &dsataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceId),
	}
	result.AbstractService.Service = result
	return result
}
