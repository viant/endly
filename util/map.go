package util

import (
	"github.com/viant/toolbox"
)

//NormalizeMap normalizes keyValuePairs from map or slice (map with preserved key order)
func NormalizeMap(keyValuePairs interface{}, deep bool) (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	if keyValuePairs == nil {
		return result, nil
	}

	err:= toolbox.ProcessMap(keyValuePairs, func(key, value interface{}) bool {
		result[toolbox.AsString(key)] = value
		if deep {
			if value != nil && toolbox.IsSlice(value) { //yaml style map conversion if applicable
				aMap, err := NormalizeMap(value, deep)
				if err != nil {
					return true
				}
				result[toolbox.AsString(key)] = aMap
			}
		}
		return true
	});

	return result, err
}

//Params represents parameters
type Params map[string]interface{}

//AppendMap source to dest map
func Append(source, dest map[string]interface{}, override bool) {
	for k, v := range source {
		if _, ok := dest[k]; ok && !override {
			continue
		}
		dest[k] = v
	}
}
