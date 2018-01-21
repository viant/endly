package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"regexp"
	"strings"
)

//ValidationIndexByDirective represent indexing directive
const ValidationIndexByDirective = "@indexBy@"

//ValidateTimeFormatDirective time format directive
const ValidateTimeFormatDirective = "@timeFormat@"

//Validator represents a validator
type Validator struct {
	ExcludedFields    map[string]bool
	TimeFormat        map[string]string
	DefaultTimeFormat string
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
	if toolbox.IsSlice(actual) {
		if toolbox.IsMap(expected) { //convert actual slice to map using expected indexBy directive
			expectedMap := toolbox.AsMap(expected)
			if indexField, ok := expectedMap[ValidationIndexByDirective]; ok {
				var actualMap = make(map[string]interface{})
				actualMap[ValidationIndexByDirective] = indexField
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

		if !toolbox.IsSlice(expected) {
			assertionInfo.AddFailure(NewFailedTest(path, fmt.Sprintf("incompatible types, expected %T but had %v", expected, actual), expected, actual))
			return nil
		}

		err := s.assertSlice(toolbox.AsSlice(expected), toolbox.AsSlice(actual), assertionInfo, path)
		if err != nil {
			return err
		}
		return nil

	}
	if toolbox.IsMap(actual) {
		if !toolbox.IsMap(expected) {
			assertionInfo.AddFailure(NewFailedTest(path, fmt.Sprintf("incompatible types, expected %T but had %v", expected, actual), expected, actual))
			return nil
		}
		err := s.assertMap(toolbox.AsMap(expected), toolbox.AsMap(actual), assertionInfo, path)
		if err != nil {
			return err
		}
		return nil
	}
	expectedText := strings.TrimSpace(toolbox.AsString(expected))
	actualText := strings.TrimSpace(toolbox.AsString(actual))

	if s.assertJSONIfConvertible(expectedText, actualText, assertionInfo, path) {
		return nil
	}

	if toolbox.IsTime(expected) {

		expectedTime, _ := toolbox.ToTime(expected, s.DefaultTimeFormat)
		actualTime, err := toolbox.ToTime(actual, s.DefaultTimeFormat)
		if err == nil {
			if expectedTime.UnixNano() == actualTime.UnixNano() {
				return nil
			}
		}

	}

	s.assertText(expectedText, actualText, assertionInfo, path)
	return nil
}

//indexLines returns index or nil if at least one entry does not have index value
func (s *Validator) indexLines(indexBy string, lines []string) map[string]map[string]interface{} {
	var result = make(map[string]map[string]interface{})
	for _, line := range lines {
		aMap, err := toolbox.JSONToMap(line)
		if err != nil {
			return nil
		}
		if indexValue, has := aMap[indexBy]; has {
			result[toolbox.AsString(indexValue)] = aMap
		} else {
			return nil
		}
	}
	return result
}

func (s *Validator) assertIndexableJSON(indexBy string, expectedLines []string, actualLines []string, assertionInfo *ValidationInfo, path string) bool {
	expectedIndex := s.indexLines(indexBy, expectedLines)
	actualIndex := s.indexLines(indexBy, actualLines)
	if expectedIndex == nil || actualIndex == nil {
		return false
	}
	err := s.Assert(expectedIndex, actualIndex, assertionInfo, path)
	return err == nil
}

func (s *Validator) extractTimeFormat(aMap map[string]interface{}) {
	for k, v := range aMap {
		if strings.HasPrefix(k, ValidateTimeFormatDirective) {
			if len(s.TimeFormat) == 0 {
				s.TimeFormat = make(map[string]string)
			}
			var field = strings.Replace(k, ValidateTimeFormatDirective, "", 1)
			s.TimeFormat[field] = toolbox.AsString(v)
			s.DefaultTimeFormat = toolbox.AsString(v)
		}
	}
}

func (s *Validator) assertJSONIfConvertible(expectedText string, actualText string, assertionInfo *ValidationInfo, path string) bool {
	if toolbox.IsCompleteJSON(expectedText) {
		if toolbox.IsNewLineDelimitedJSON(expectedText) {
			expectedLines := strings.Split(expectedText, "\n")
			actualLines := strings.Split(actualText, "\n")
			if aMap, err := toolbox.JSONToMap(expectedLines[0]); err == nil {
				s.extractTimeFormat(aMap)
				if index, ok := aMap[ValidationIndexByDirective]; ok {
					expectedLines = expectedLines[1:]
					return s.assertIndexableJSON(toolbox.AsString(index), expectedLines, actualLines, assertionInfo, path)
				}
			}

			if len(expectedLines) != len(actualLines) {
				assertionInfo.AddFailure(NewFailedTest(path, fmt.Sprintf("missing lines, expected %v but had %v", len(expectedLines), len(actualLines)), len(expectedLines), len(actualLines)))
			}
			var length = len(expectedLines)
			if length > len(actualLines) {
				length = len(actualLines)
			}

			for i := 0; i < length; i++ {
				_ = s.Assert(expectedLines[i], actualLines[i], assertionInfo, fmt.Sprintf("[%v]", i))
			}
			return true

		}
		if expectedMap, err := toolbox.JSONToMap(expectedText); err == nil {
			if actualMap, err := toolbox.JSONToMap(actualText); err == nil {
				if err = s.assertMap(expectedMap, actualMap, assertionInfo, path); err == nil {
					return true
				}
			}
		}
	}
	return false
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
	if len(s.TimeFormat) == 0 {
		s.TimeFormat = make(map[string]string)
	}
	s.extractTimeFormat(expectedMap)

	for key, expected := range expectedMap {
		if strings.HasPrefix(key, ValidateTimeFormatDirective) {
			continue
		}
		if s.ExcludedFields[key] {
			continue
		}
		if format, ok := s.TimeFormat[key]; ok {
			timeValue, err := toolbox.ToTime(expected, toolbox.DateFormatToLayout(format))
			if err == nil {
				expected = timeValue
				expectedMap[key] = expected
			}
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

		if format, ok := s.TimeFormat[key]; ok {
			timeValue, err := toolbox.ToTime(actual, toolbox.DateFormatToLayout(format))
			if err == nil {
				actual = timeValue
			}
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
