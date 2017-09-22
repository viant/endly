package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"unicode"
)

const (
	expectVariableStart = iota
	expectVariableName
	expectVariableNameEnclosureEnd
)

func Expand(state common.Map, text string) string {
	if strings.Index(text, "$") == -1 {
		return text
	}
	var expandVariable = func(variableName, result string) string {

		value, has := state.GetValue(string(variableName[1:]))
		if has {
			return result + toolbox.AsString(value)
		}
		return result + variableName
	}
	var variableName = ""
	var parsingState = expectVariableStart
	var result = ""

	for i, rune := range text {
		aChar := string(text[i : i+1])
		switch parsingState {
		case expectVariableStart:
			if aChar == "$" {
				variableName += aChar
				if i+1 < len(text) {
					nextChar := string(text[i+1 : i+2])
					if nextChar == "{" {
						parsingState = expectVariableNameEnclosureEnd
						continue

					}
				}
				parsingState = expectVariableName
				continue
			}
			result += aChar

		case expectVariableNameEnclosureEnd:
			variableName += aChar
			if aChar != "}" {
				continue
			}
			result = expandVariable(variableName, result)
			variableName = ""
			parsingState = expectVariableStart

		case expectVariableName:
			if unicode.IsLetter(rune) || unicode.IsDigit(rune) || aChar == "." || aChar == "_" {
				variableName += aChar
				continue
			}
			result = expandVariable(variableName, result)
			result += aChar
			variableName = ""
			parsingState = expectVariableStart

		}
	}
	if len(variableName) > 0 {
		result = expandVariable(variableName, result)
	}
	return result
}

func ExpandValue(source interface{}, state common.Map) interface{} {
	switch value := source.(type) {
	case string:
		if strings.HasPrefix(value, "$") {
			if state.Has(string(value[1:])) {
				return state.Get(string(value[1:]))
			}
		}
		return Expand(state, value)
	case map[string]interface{}:
		var resultMap = make(map[string]interface{})
		for k, v := range value {
			resultMap[Expand(state, k)] = ExpandValue(v, state)
		}
		return resultMap
	case []interface{}:
		var resultSlice = make([]interface{}, len(value))
		for i, value := range value {
			resultSlice[i] = ExpandValue(value, state)
		}
		return resultSlice
	default:
		if toolbox.IsMap(source) {
			return ExpandValue(toolbox.AsMap(value), state)
		} else if toolbox.IsSlice(source) {
			return ExpandValue(toolbox.AsSlice(value), state)
		} else {
			return ExpandValue(toolbox.AsString(value), state)
		}
	}
	return source
}

func ExpandAsMap(source interface{}, state common.Map) (map[string]interface{}, error) {
	var candidate = ExpandValue(source, state)
	if result, ok := candidate.(map[string]interface{}); ok {
		return result, nil
	}
	available := toolbox.MapKeysToStringSlice(state)
	return nil, fmt.Errorf("Expected a map but had %T in '%v', avaiable var [%v]", source, source, strings.Join(available, ","))
}
