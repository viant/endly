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
	err := toolbox.ProcessMap(keyValuePairs, func(k, value interface{}) bool {
		var key = toolbox.AsString(k)
		//inline map key
		result[key] = value
		if deep {
			if normalized, err := toolbox.NormalizeKVPairs(value); err == nil {
				result[key] = normalized
			}

		}
		return true
	})
	return result, err
}

//AppendMap source to dest map
func Append(dest, source map[string]interface{}, override bool) {
	for k, v := range source {
		if _, ok := dest[k]; ok && !override {
			continue
		}
		dest[k] = v
	}
}
