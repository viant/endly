package udf

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/viant/toolbox/data"
	tudf "github.com/viant/toolbox/data/udf"
)

// AsMap decodes JSON objects without YAML/key-value normalization, which can
// turn JSON arrays into objects. Non-JSON inputs retain the native AsMap behavior.
func AsMap(source interface{}, state data.Map) (interface{}, error) {
	var text string
	switch value := source.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return tudf.AsMap(source, state)
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var result map[string]interface{}
	if err := decoder.Decode(&result); err != nil {
		return tudf.AsMap(source, state)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("AsMap requires one JSON object")
	}
	return nativeJSONNumbers(result)
}

// Native workflow expansion handles Go numeric types; json.Number is a named
// string and would otherwise be expanded as text.
func nativeJSONNumbers(value interface{}) (interface{}, error) {
	switch actual := value.(type) {
	case json.Number:
		text := actual.String()
		if !strings.ContainsAny(text, ".eE") {
			if signed, err := strconv.ParseInt(text, 10, 64); err == nil {
				return signed, nil
			}
			return strconv.ParseUint(text, 10, 64)
		}
		return strconv.ParseFloat(text, 64)
	case map[string]interface{}:
		for key, item := range actual {
			number, err := nativeJSONNumbers(item)
			if err != nil {
				return nil, err
			}
			actual[key] = number
		}
	case []interface{}:
		for i, item := range actual {
			number, err := nativeJSONNumbers(item)
			if err != nil {
				return nil, err
			}
			actual[i] = number
		}
	}
	return value, nil
}
