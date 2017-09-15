package endly


const ValidatorServiceId = "transfer"

type validatorService struct {
	*AbstractService
}


type AssertRequest struct {

}


func (s *validatorService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}

	switch actualReuest := request.(type) {
	case *AssertRequest:
		s.assert(actualReuest)

	}
	return response
}


func (service *validatorService) assert(request *AssertRequest) {


}

func (s *validatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "assert":
		return &AssertRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewValidatorService() Service {
	var result = &validatorService{
		AbstractService: NewAbstractService(TransferServiceId),
	}
	result.AbstractService.Service = result
	return result

}

