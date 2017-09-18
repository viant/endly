package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"reflect"
	"regexp"
	"strings"
)

const ValidatorServiceId = "validator"

type ValidatorService struct {
	*AbstractService
}

type ValidatorAssertRequest struct {
	Actual   interface{}
	Expected interface{}
}

type ValidatorAssertResponse struct {
	TestPassed int
	TestFailed []string
}

func (ar *ValidatorAssertResponse) AddFailure(message string) {
	if len(ar.TestFailed) == 0 {
		ar.TestFailed = make([]string, 0)
	}
	ar.TestFailed = append(ar.TestFailed, message)
}

func (ar *ValidatorAssertResponse) HasFailure() bool {
	return len(ar.TestFailed) > 0
}

func (ar *ValidatorAssertResponse) Message() string {
	return fmt.Sprintf("Passed: %v\nFailed:%v\n-----\n\t%v\n",
		ar.TestPassed,
		len(ar.TestFailed),
		strings.Join(ar.TestFailed, "\n\t"),
	)
}

func (s *ValidatorService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	switch actualReuest := request.(type) {
	case *ValidatorAssertRequest:
		assertResponse, err := s.Assert(context, actualReuest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
		response.Response = assertResponse
		if assertResponse.HasFailure() {
			response.Error = assertResponse.Message()
		}
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *ValidatorService) Assert(context *Context, request *ValidatorAssertRequest) (*ValidatorAssertResponse, error) {
	var response = &ValidatorAssertResponse{}
	var state = context.State()
	var actual = request.Actual
	var expected = request.Expected

	if toolbox.IsString(request.Actual) {
		if actualValue, ok := state.GetValue(toolbox.AsString(request.Actual)); ok {
			actual = actualValue
		}
	}
	err := s.assertValue(expected, actual, response, "/")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *ValidatorService) assertValue(expected, actual interface{}, response *ValidatorAssertResponse, path string) error {

	if toolbox.IsValueOfKind(actual, reflect.Slice) {

		if toolbox.IsValueOfKind(expected, reflect.Map) { //convert actual slice to map using expected indexBy directive
			expectedMap := toolbox.AsMap(expected)
			if indexField, ok := expectedMap["@indexBy@"]; ok {
				var actualMap = make(map[string]interface{})
				actualMap["@indexBy@"] = indexField
				var actualSlice = toolbox.AsSlice(actual)
				for _, item := range actualSlice {
					var itemMap = toolbox.AsMap(item)
					if key, has := itemMap[toolbox.AsString(indexField)]; has {
						actualMap[toolbox.AsString(key)] = itemMap
					}
				}
				return s.assertValue(expected, actualMap, response, path)
			}
		}

		if !toolbox.IsValueOfKind(expected, reflect.Slice) {
			response.AddFailure(fmt.Sprintf("Incompatbile types, expected %T but had %v", expected, actual))
			return nil
		}

		err := s.assertSlice(toolbox.AsSlice(expected), toolbox.AsSlice(actual), response, path)
		if err != nil {
			return err
		}
		return nil

	}
	if toolbox.IsValueOfKind(actual, reflect.Map) {
		if !toolbox.IsValueOfKind(expected, reflect.Map) {
			response.AddFailure(fmt.Sprintf("Incompatbile types, expected %T but had %v", expected, actual))
			return nil
		}
		err := s.assertMap(toolbox.AsMap(expected), toolbox.AsMap(actual), response, path)
		if err != nil {
			return err
		}
		return nil
	}
	expectedText := toolbox.AsString(expected)
	actualText := toolbox.AsString(actual)
	s.assertText(expectedText, actualText, response, path)
	return nil
}

func (s *ValidatorService) assertText(expected, actual string, response *ValidatorAssertResponse, path string) error {
	isRegExpr := strings.HasPrefix(expected, "~/") && strings.HasSuffix(expected, "/")
	isContains := strings.HasPrefix(expected, "/") && strings.HasSuffix(expected, "/")

	if !isRegExpr && !isContains {
		if expected != actual {
			response.AddFailure(fmt.Sprintf("actual(%T):  '%v' was not equal (%T) '%v' in path '%v'", actual, actual, expected, expected, path))
		}
		response.TestPassed++
		return nil
	}

	if isContains {
		expected = string(expected[1 : len(expected)-1])
		isReversed := strings.HasPrefix(expected, "!")
		if isReversed {
			expected = string(expected[1:])
		}

		var doesContain = strings.Contains(actual, expected)
		if !doesContain && !isReversed {
			response.AddFailure(fmt.Sprintf("actual '%v' does not contain: '%v' in path '%v'", actual, expected, path))
		} else if isReversed && doesContain {
			response.AddFailure(fmt.Sprintf("actual '%v' shold not contain: '%v' in path '%v'", actual, expected, path))
		}
		response.TestPassed++
		return nil
	}

	expected = string(expected[2 : len(expected)-1])
	isReversed := strings.HasPrefix(expected, "!")
	if isReversed {
		expected = string(expected[1:])
	}
	useMultiLine := strings.Index(actual, "\n")
	pattern := ""
	if useMultiLine > 0 {
		pattern = "?m:"
	}
	pattern += expected
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("Failed to validate '%v' and '%v', in path '%v' due to %v", expected, actual, pattern, path, err)
	}
	var matches = compiled.Match(([]byte)(actual))

	if !matches && !isReversed {
		response.AddFailure(fmt.Sprintf("actual: '%v' was not matched %v in path '%v'", actual, expected, path))
	} else if matches && isReversed {
		response.AddFailure(fmt.Sprintf("actual: '%v' should not be matched %v in path '%v'", actual, expected, path))
	}
	response.TestPassed++
	return nil
}

func (s *ValidatorService) assertMap(expectedMap map[string]interface{}, actualMap map[string]interface{}, response *ValidatorAssertResponse, path string) error {
	for key, expected := range expectedMap {
		keyPath := fmt.Sprintf("%v[%v]", path, key)
		actual, ok := actualMap[key]
		if !ok {
			response.AddFailure(fmt.Sprintf("%v was missing", keyPath))
			continue
		}
		if toolbox.AsString(expected) == "@exists@" {
			response.TestPassed++
			continue
		}
		if toolbox.AsString(expected) == "@!exists@" {
			response.AddFailure(fmt.Sprintf("'%v' should not exists but was present: %v", keyPath, actual))
			continue
		}

		err := s.assertValue(expected, actual, response, keyPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ValidatorService) assertSlice(expectedSlice []interface{}, actualSlice []interface{}, response *ValidatorAssertResponse, path string) error {
	for index, expected := range expectedSlice {
		keyPath := fmt.Sprintf("%v[%v]", path, index)
		if !(index < len(actualSlice)) {
			response.AddFailure(fmt.Sprintf("[%v+] were missing, expected size: %v, actual size: %v", keyPath, len(expectedSlice), len(actualSlice)))
			return nil
		}
		actual := actualSlice[index]
		err := s.assertValue(expected, actual, response, keyPath)
		if err != nil {
			return err
		}
	}
	return nil
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
