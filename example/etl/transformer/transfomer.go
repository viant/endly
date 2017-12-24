package transformer

import "github.com/viant/toolbox"

//Transformer represents transformer function
type Transformer func(source map[string]interface{}) ([]map[string]interface{}, error)

//Transformers represents transformer registry
var Transformers = make(map[string]Transformer)

func init() {
	Transformers["MapToSlice"] = MapToSlice
}

//MapToSlice converts map to slice
func MapToSlice(source map[string]interface{}) ([]map[string]interface{}, error) {
	var response = make(map[string]interface{})
	for k, v := range source {
		if toolbox.IsMap(v) {
			response[k] = mapToSlice(v)
			continue
		}
		response[k] = v
	}
	return []map[string]interface{}{response}, nil
}

//KeyValue represents a map entry
type KeyValue struct {
	Key   string
	Value interface{}
}

func mapToSlice(source interface{}) []interface{} {
	var result = make([]interface{}, 0)
	for k, v := range toolbox.AsMap(source) {
		if toolbox.IsMap(v) {
			v = mapToSlice(v)
		} else if toolbox.IsSlice(v) {
			aSlice := toolbox.AsSlice(v)
			for i, item := range toolbox.AsSlice(v) {
				if toolbox.IsMap(item) {
					aSlice[i] = mapToSlice(item)
				}
			}
		}
		var keyValue = &KeyValue{Key: k, Value: v}
		result = append(result, keyValue)
	}
	return result
}
