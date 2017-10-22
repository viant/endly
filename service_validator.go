package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

const ValidatorServiceId = "validator"

type ValidatorService struct {
	*AbstractService
}



type ValidatorAssertRequest struct {
	Name string
	Actual   interface{}
	Expected interface{}
}


type ValidatorAssertionInfo struct {
	TestPassed int
	TestFailed []string
}

func (ar *ValidatorAssertionInfo) AddFailure(message string) {
	if len(ar.TestFailed) == 0 {
		ar.TestFailed = make([]string, 0)
	}
	ar.TestFailed = append(ar.TestFailed, message)
}

func (ar *ValidatorAssertionInfo) HasFailure() bool {
	return len(ar.TestFailed) > 0
}

func (ar *ValidatorAssertionInfo) Message() string {
	return fmt.Sprintf("Passed: %v\nFailed:%v\n-----\n\t%v\n",
		ar.TestPassed,
		len(ar.TestFailed),
		strings.Join(ar.TestFailed, "\n\t"),
	)
}

func (s *ValidatorService) Run(context *Context, request interface{}) *ServiceResponse {
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

func (s *ValidatorService) Assert(context *Context, request *ValidatorAssertRequest) (*ValidatorAssertionInfo, error) {
	var response = &ValidatorAssertionInfo{}
	var state = context.State()
	var actual = request.Actual
	var expected = request.Expected
	if toolbox.IsString(request.Actual) {
		if actualValue, ok := state.GetValue(toolbox.AsString(request.Actual)); ok {
			actual = actualValue
		}
	}
	validator := &Validator{
		SkipFields: make(map[string]bool),
	}
	err := validator.Assert(expected, actual, response, "/")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *ValidatorService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "assert":
		return &ValidatorAssertRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewValidatorService() Service {
	var result = &ValidatorService{
		AbstractService: NewAbstractService(ValidatorServiceId),
	}
	result.AbstractService.Service = result
	return result

}
