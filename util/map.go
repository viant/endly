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
		result[key] = value
		if deep {
			if value == nil {
				return true
			}
			if toolbox.IsMap(value) {
				if normalized, err := NormalizeMap(value, deep); err == nil {
					result[key] = normalized
				}
			} else if toolbox.IsSlice(value) { //yaml style map conversion if applicable
				aSlice := toolbox.AsSlice(value)
				if len(aSlice) == 0 {
					return true
				}
				if toolbox.IsMap(aSlice[0]) || toolbox.IsStruct(aSlice[0]) {
					normalized, err := NormalizeMap(value, deep)
					if err == nil {
						result[key] = normalized
					}
				} else if toolbox.IsSlice(aSlice[0]) {
					for i, item := range aSlice {
						itemMap, err := NormalizeMap(item, deep)
						if err != nil {
							return true
						}
						aSlice[i] = itemMap
					}
					result[key] = aSlice
				}
				return true
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
