package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"reflect"
	"regexp"
	"strings"
)

//Validator represents a validator
type Validator struct {
	ExcludedFields map[string]bool
}

//Check checks expected vs actual value, and returns true if all assertion passes.
func (s *Validator) Check(expected, actual interface{}) (bool, error) {
	var response = &ValidationInfo{}
	err := s.Assert(expected, actual, response, "")
	if err != nil {
		return false, err
	}
	return !response.HasFailure(), nil
}

//Assert check if actual matches expected value, in any case it update assert info with provided validation path.
func (s *Validator) Assert(expected, actual interface{}, assertionInfo *ValidationInfo, path string) error {
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
				return s.Assert(expected, actualMap, assertionInfo, path)
			}
		}

		if !toolbox.IsValueOfKind(expected, reflect.Slice) {
			assertionInfo.AddFailure(NewFailedTest(path, fmt.Sprintf("Incompatbile types, expected %T but had %v", expected, actual), expected, actual))
			return nil
		}

		err := s.assertSlice(toolbox.AsSlice(expected), toolbox.AsSlice(actual), assertionInfo, path)
		if err != nil {
			return err
		}
		return nil

	}
	if toolbox.IsValueOfKind(actual, reflect.Map) {
		if !toolbox.IsValueOfKind(expected, reflect.Map) {
			assertionInfo.AddFailure(NewFailedTest(path, fmt.Sprintf("Incompatbile types, expected %T but had %v", expected, actual), expected, actual))
			return nil
		}
		err := s.assertMap(toolbox.AsMap(expected), toolbox.AsMap(actual), assertionInfo, path)
		if err != nil {
			return err
		}
		return nil
	}
	expectedText := toolbox.AsString(expected)
	actualText := toolbox.AsString(actual)


	s.assertText(expectedText, actualText, assertionInfo, path)
	return nil
}

func (s *Validator) assertEqual(expected, actual string, response *ValidationInfo, path string) error {
	isReversed := strings.HasPrefix(expected, "!")
	if isReversed {
		expected = string(expected[1:])
	}

	if expected != actual && !isReversed {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual(%T):  '%v' was not equal (%T) '%v'", actual, actual, expected, expected), expected, actual))
		return nil
	}
	if expected == actual && isReversed {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual(%T):  '%v' was not equal (%T) '%v'", actual, actual, expected, expected), expected, actual))
		return nil
	}
	response.TestPassed++
	return nil
}

func (s *Validator) assertRange(isReversed bool, expected, actual string, response *ValidationInfo, path string) error {
	expected = string(expected[1 : len(expected)-1])
	if strings.Contains(expected, "..") {
		var rangeValue = strings.Split(expected, "..")
		var minExpected = toolbox.AsFloat(rangeValue[0])
		var maxExpected = toolbox.AsFloat(rangeValue[1])
		var actualNumber = toolbox.AsFloat(actual)

		if actualNumber >= minExpected && actualNumber <= maxExpected && !isReversed {
			response.TestPassed++
			return nil
		}
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual '%v' is not between'%v and %v'", actual, minExpected, maxExpected), minExpected, actual))

	} else if strings.Contains(expected, ",") {
		var alternatives = strings.Split(expected, ",")
		var doesContain = false
		for _, expectedCandidate := range alternatives {
			if strings.Contains(actual, expectedCandidate) {
				doesContain = true
				break
			}
		}
		if !doesContain && !isReversed {
			response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual '%v' does not contain: '%v'", actual, alternatives), alternatives, actual))
		} else if isReversed && doesContain {
			response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual '%v' shold not contain: '%v'", actual, alternatives), alternatives, actual))
		}
		response.TestPassed++
	}
	return nil
}

func (s *Validator) assertContains(expected, actual string, response *ValidationInfo, path string) error {
	expected = string(expected[1 : len(expected)-1])
	isReversed := strings.HasPrefix(expected, "!")
	if isReversed {
		expected = string(expected[1:])
	}
	if strings.HasPrefix(expected, "[") && strings.HasSuffix(expected, "]") {
		return s.assertRange(isReversed, expected, actual, response, path)
	}
	var doesContain = strings.Contains(actual, expected)
	if !doesContain && !isReversed {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual '%v' does not contain: '%v'", actual, expected), actual, expected))
	} else if isReversed && doesContain {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual '%v' shold not contain: '%v'", actual, expected), actual, expected))
	}
	response.TestPassed++
	return nil
}

func (s *Validator) assertRegExpr(expected, actual string, response *ValidationInfo, path string) error {
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
		return fmt.Errorf("failed to validate '%v' and '%v' with pattern: %v, due to %v", expected, actual, pattern, err)
	}
	var matches = compiled.Match(([]byte)(actual))

	if !matches && !isReversed {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual: '%v' was not matched %v", actual, expected), expected, actual))
	} else if matches && isReversed {
		response.AddFailure(NewFailedTest(path, fmt.Sprintf("actual: '%v' should not be matched %v", actual, expected), expected, actual))
	}
	response.TestPassed++
	return nil
}

func (s *Validator) assertText(expected, actual string, response *ValidationInfo, path string) error {
	isRegExpr := strings.HasPrefix(expected, "~/") && strings.HasSuffix(expected, "/")
	isContains := strings.HasPrefix(expected, "/") && strings.HasSuffix(expected, "/")
	var isEqual = !isRegExpr && !isContains


	if isEqual {
		return s.assertEqual(expected, actual, response, path)
	}
	if isContains {
		return s.assertContains(expected, actual, response, path)
	}

	return s.assertRegExpr(expected, actual, response, path)
}

func (s *Validator) assertMap(expectedMap map[string]interface{}, actualMap map[string]interface{}, response *ValidationInfo, path string) error {
	for key, expected := range expectedMap {
		if s.ExcludedFields[key] {
			continue
		}
		keyPath := fmt.Sprintf("%v[%v]", path, key)
		actual, ok := actualMap[key]
		if !ok {
			response.AddFailure(NewFailedTest(path, fmt.Sprintf("%v was missing", keyPath), expected, actual))

			continue
		}
		if toolbox.AsString(expected) == "@exists@" {
			response.TestPassed++
			continue
		}
		if toolbox.AsString(expected) == "@!exists@" {
			response.AddFailure(NewFailedTest(path, fmt.Sprintf("'%v' should not exists but was present: %v", keyPath, actual), expected, actual))
			continue
		}

		err := s.Assert(expected, actual, response, keyPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Validator) assertSlice(expectedSlice []interface{}, actualSlice []interface{}, response *ValidationInfo, path string) error {
	for index, expected := range expectedSlice {
		keyPath := fmt.Sprintf("%v[%v]", path, index)
		if !(index < len(actualSlice)) {
			response.AddFailure(NewFailedTest(keyPath, fmt.Sprintf("expected size: %v, actual size: %v", len(expectedSlice), len(actualSlice)), len(expectedSlice), len(actualSlice)))
			return nil
		}
		actual := actualSlice[index]
		err := s.Assert(expected, actual, response, keyPath)
		if err != nil {
			return err
		}
	}
	return nil
}
