package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

//ValidatorServiceID validator service id
const ValidatorServiceID = "validator"

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
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *validatorService) Assert(context *Context, request *ValidatorAssertRequest) (*ValidationInfo, error) {
	var response = &ValidationInfo{
		Description: request.Description,
	}
	var state = context.State()
	var actual = request.Actual
	var expected = request.Expected
	if toolbox.IsString(request.Actual) {
		if actualValue, ok := state.GetValue(toolbox.AsString(request.Actual)); ok {
			actual = actualValue
		}
	}
	validator := &Validator{
		ExcludedFields: make(map[string]bool),
	}
	err := validator.Assert(expected, actual, response, "/")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *validatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "assert":
		return &ValidatorAssertRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewValidatorService creates a new validation service
func NewValidatorService() Service {
	var result = &validatorService{
		AbstractService: NewAbstractService(ValidatorServiceID),
	}
	result.AbstractService.Service = result
	return result

}
