package util

import "github.com/viant/toolbox"

//NormalizeMap normalizes parameters
func NormalizeMap(params interface{}) (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	if params == nil {
		return result, nil
	}
	err := toolbox.ProcessMap(params, func(key, value interface{}) bool {
		result[toolbox.AsString(key)] = value
		return true
	})
	return result, err
}
