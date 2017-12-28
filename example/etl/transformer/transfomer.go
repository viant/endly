package transformer

import (
	"fmt"
	"github.com/viant/toolbox"
)

//Transformer represents transformer function
type Transformer func(source map[string]interface{}) ([]map[string]interface{}, error)

//Transformers represents transformer registry
var Transformers = make(map[string]Transformer)

func init() {
	Transformers["Flatten"] = Flatten
}

//MapToSlice converts map to slice
func Flatten(source map[string]interface{}) ([]map[string]interface{}, error) {
	var result = make([]map[string]interface{}, 0)

	var recordProvider = func() map[string]interface{} {
		var result = make(map[string]interface{})
		for k, v := range source {
			if toolbox.IsMap(v) || toolbox.IsSlice(v) {
				continue
			}
			result[k] = v
		}
		return result
	}

	for k, v := range source {
		if toolbox.IsSlice(v) {
			for _, item := range toolbox.AsSlice(v) {
				var record = recordProvider()
				var itemMap = toolbox.AsMap(item)
				for j, v := range itemMap {
					record[fmt.Sprintf("%v_%v", k, j)] = v
				}
				result = append(result, record)
			}
		}
	}
	if len(result) == 0 {
		var record = recordProvider()
		result = append(result, record)
	}
	return result, nil
}
