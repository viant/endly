package endly

import (
	"bytes"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"time"
	"unicode"
)

const (
	expectVariableStart = iota
	expectVariableName
	expectVariableNameEnclosureEnd
)

func ExpandAsText(state common.Map, text string) string {
	result := Expand(state, text)
	if toolbox.IsSlice(result) || toolbox.IsMap(result) {
		buf := new(bytes.Buffer)
		err := toolbox.NewJSONEncoderFactory().Create(buf).Encode(result)
		if err == nil {
			return buf.String()
		}
	}
	return toolbox.AsString(result)
}

func asExpandedText(source interface{}) string {
	if toolbox.IsSlice(source) || toolbox.IsMap(source) {
		buf := new(bytes.Buffer)
		err := toolbox.NewJSONEncoderFactory().Create(buf).Encode(source)
		if err == nil {
			return buf.String()
		}
	}
	return toolbox.AsString(source)
}

func Expand(state common.Map, text string) interface{} {
	if strings.Index(text, "$") == -1 {
		return text
	}
	var expandVariable = func(variableName string) interface{} {
		value, has := state.GetValue(string(variableName[1:]))
		if has {
			if value != nil && (toolbox.IsMap(value) || toolbox.IsSlice(value)) {
				return ExpandValue(value, state)
			}
			return value
		}
		return variableName
	}

	var variableName = ""
	var parsingState = expectVariableStart
	var result = ""

	for i, rune := range text {
		aChar := string(text[i : i+1])
		var isLast = i+1 == len(text)
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
			var expanded = expandVariable(variableName)
			if isLast && result == "" {
				return expanded
			}
			result += asExpandedText(expanded)
			variableName = ""
			parsingState = expectVariableStart

		case expectVariableName:
			if unicode.IsLetter(rune) || unicode.IsDigit(rune) || aChar == "." || aChar == "_" || aChar == "+" || aChar == "<" || aChar == "-" {
				variableName += aChar
				continue
			}
			var expanded = expandVariable(variableName)
			if isLast && result == "" {
				return expanded
			}
			result += asExpandedText(expanded)
			result += aChar
			variableName = ""
			parsingState = expectVariableStart

		}
	}
	if len(variableName) > 0 {
		var expanded = expandVariable(variableName)
		if result == "" {
			return expanded
		}
		result += asExpandedText(expanded)
	}

	return result
}

func ExpandValue(source interface{}, state common.Map) interface{} {

	switch value := source.(type) {
	case bool, []byte, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64, time.Time:
		return source
	case string:

		udf, value, _ := getUdfIfDefined(value)
		if strings.HasPrefix(value, "$") {
			_, has := state.GetValue(string(value[1:]))
			result := Expand(state, value)
			if has && udf != nil {
				transformed, err := udf(result, state)
				if err == nil {
					result = transformed
				}
			}
			return result
		} else if udf != nil {
			transformed, err := udf(value, state)
			if err == nil {
				return transformed
			}
		}
		return ExpandAsText(state, value)
	case map[string]interface{}:
		var resultMap = make(map[string]interface{})
		for k, v := range value {
			var expanded = ExpandValue(v, state)
			resultMap[ExpandAsText(state, k)] = expanded
		}
		return resultMap
	case []interface{}:
		var resultSlice = make([]interface{}, len(value))
		for i, value := range value {
			resultSlice[i] = ExpandValue(value, state)
		}
		return resultSlice
	default:
		if source == nil {
			return nil
		}

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
