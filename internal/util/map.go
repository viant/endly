package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

// NormalizeMap normalizes keyValuePairs from map or slice (map with preserved key order)
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
			if normalized, err := normalizeKVPairs(value); err == nil {
				result[key] = normalized
			}

		}
		return true
	})
	return result, err
}

// normalizeKVPairs retains the difference between an empty YAML/JSON list and
// an empty object. toolbox.NormalizeKVPairs treats an empty slice as a map of
// zero key-value pairs, which changes an expanded expectation such as roles: []
// into roles: {}. Nonempty key-value lists keep their existing normalization.
func normalizeKVPairs(source interface{}) (interface{}, error) {
	if source == nil {
		return nil, nil
	}
	if toolbox.IsSlice(source) && len(toolbox.AsSlice(source)) == 0 {
		return source, nil
	}
	if !toolbox.IsMap(source) && !toolbox.IsStruct(source) && !toolbox.IsSlice(source) {
		return source, nil
	}
	aMap := make(map[string]interface{})
	err := toolbox.ProcessMap(source, func(k, value interface{}) bool {
		key := toolbox.AsString(k)
		aMap[key] = value
		if normalized, normalizeErr := normalizeKVPairs(value); normalizeErr == nil {
			aMap[key] = normalized
		}
		return true
	})
	if err == nil {
		return aMap, nil
	}
	if toolbox.IsSlice(source) {
		items := toolbox.AsSlice(source)
		for i, item := range items {
			if normalized, normalizeErr := normalizeKVPairs(item); normalizeErr == nil {
				items[i] = normalized
			}
		}
		return items, nil
	}
	return source, err
}

// Append source to dest map
func Append(dest, source map[string]interface{}, override bool) {
	for k, v := range source {
		if _, ok := dest[k]; ok && !override {
			continue
		}
		dest[k] = v
	}
}

// MergeMap merges source map into dest map
func MergeMap(dest, source map[string]interface{}) {
	for k, v := range source {
		if destValue, ok := dest[k]; ok {
			if destMap, ok := destValue.(data.Map); ok {
				if sourceMap, ok := v.(data.Map); ok {
					MergeMap(destMap, sourceMap)
					continue
				}
			}
		}
		dest[k] = v
	}
}

// BuildLowerCaseMapping build lowercase key to key map mapping
func BuildLowerCaseMapping(aMap map[string]interface{}) map[string]string {
	var result = make(map[string]string)
	for k := range aMap {
		result[strings.ToLower(k)] = k
	}
	return result
}
