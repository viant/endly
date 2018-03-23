package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

func AsDataMap(source interface{}) data.Map {
	if source == nil {
		return data.Map(map[string]interface{}{})
	}
	var aMap = make(map[string]interface{})
	if toolbox.IsStruct(source) {
		toolbox.DefaultConverter.AssignConverted(&aMap, source)
	} else if toolbox.IsMap(source) {
		aMap = toolbox.AsMap(source)
	}
	return data.Map(aMap)
}

//AsExtractable returns  text and data structure
func AsExtractable(input interface{}) (string, map[string]interface{}) {
	var extractableOutput string
	var structuredOutput data.Map
	switch value := input.(type) {
	case string:
		extractableOutput = value
	case []byte:
		extractableOutput = string(value)
	case []interface{}:
		if len(value) > 0 {
			if toolbox.IsString(value[0]) {
				extractableOutput = toolbox.AsString(value[0])
			} else {
				structuredOutput = AsDataMap(value[0])
			}
		}
	default:
		structuredOutput = AsDataMap(value)
	}
	if extractableOutput != "" {
		if toolbox.IsCompleteJSON(extractableOutput) {
			if strings.HasPrefix(strings.Trim(extractableOutput, " \r\n"), "[") {
				structuredOutput = data.NewMap()
				if aSlice, err := toolbox.JSONToSlice(extractableOutput); err == nil {
					structuredOutput.Put("value", aSlice)
				}
			} else if aMap, err := toolbox.JSONToMap(extractableOutput); err == nil {
				structuredOutput = data.Map(aMap)
			}
		}
	}
	return extractableOutput, structuredOutput
}
