package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

//ValidatorServiceID represents validator service id
const ValidatorServiceID = "validator"

//ValidatorServiceAssertAction represents assert action
const ValidatorServiceAssertAction = "assert"

type validatorService struct {
	*AbstractService
}

func (s *validatorService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))

	switch actualReuest := request.(type) {
	case *ValidatorAssertRequest:
		assertResponse, err := s.Assert(context, actualReuest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
		response.Response = assertResponse
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *validatorService) Assert(context *Context, request *ValidatorAssertRequest) (response *ValidatorAssertResponse, err error) {
	var state = context.State()
	var actual = request.Actual
	var expected = request.Expected
	response = &ValidatorAssertResponse{}
	if toolbox.IsString(request.Actual) {
		if actualValue, ok := state.GetValue(toolbox.AsString(request.Actual)); ok {
			actual = actualValue
		}
	}
	response.Validation, err =  Assert(context, "/", expected, actual)
	if err != nil {
		return nil, err
	}
	return response, nil
}



func (s *validatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case ValidatorServiceAssertAction:
		return &ValidatorAssertRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func (s *validatorService) NewResponse(action string) (interface{}, error) {
	switch action {
	case ValidatorServiceAssertAction:
		return &ValidatorAssertResponse{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

//NewValidatorService creates a new validation service
func NewValidatorService() Service {
	var result = &validatorService{
		AbstractService: NewAbstractService(ValidatorServiceID,
			ValidatorServiceAssertAction),
	}
	result.AbstractService.Service = result
	return result

}
