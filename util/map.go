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
	err := toolbox.ProcessMap(keyValuePairs, func(key, value interface{}) bool {
		result[toolbox.AsString(key)] = value

		if deep {
			if value != nil && toolbox.IsSlice(value) { //yaml style map conversion if applicable
				aSlice := toolbox.AsSlice(value)
				if len(aSlice) > 0 {
					if toolbox.IsMap(aSlice[0]) {
						if normalized, err := NormalizeMap(value, deep);err == nil{
							result[toolbox.AsString(key)] = normalized
						}
					} else if toolbox.IsSlice(aSlice[0]) {
						for i, item := range aSlice {
							itemMap, err := NormalizeMap(item, deep)
							if err != nil {
								return true
							}
							aSlice[i] = itemMap
						}
					}
					return true
				}
			}
		}
		return true
	});
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
