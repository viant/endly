package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

const ValidatorServiceId = "validator"

type validatorService struct {
	*AbstractService
}


type AssertRequest struct {
	Actual interface{}
	Expected interface{}
}


func (s *validatorService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}


	switch actualReuest := request.(type) {
	case *AssertRequest:
		s.assert(context, actualReuest)

	}
	return response
}


func (service *validatorService) assert(context *Context, request *AssertRequest) {
	var state = context.State()
	var actual , ok  = state.GetValue(toolbox.AsString(request.Actual))
	if ! ok {
		actual = request.Actual
	}
	fmt.Printf("STATE: %v\n", state)

	fmt.Printf("EXPECTED: %v\n", request.Expected)
	fmt.Printf("ACTUAL: %v\n", actual)

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
		AbstractService: NewAbstractService(ValidatorServiceId),
	}
	result.AbstractService.Service = result
	return result

}

