package dsunit

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

// TableData represents table data
type TableData struct {
	Table         string
	Value         interface{} //deprecated, use Data attribute instead
	Data          interface{}
	AutoGenerate  map[string]string `json:",omitempty"`
	PostIncrement []string          `json:",omitempty"`
	Key           string
}

// AutoGenerateIfNeeded retrieves auto generated values
func (d *TableData) AutoGenerateIfNeeded(state data.Map) error {
	for k, v := range d.AutoGenerate {
		var value interface{}
		if v[0:2] == "${" && v[len(v)-1:] == "}" {
			value = state.Expand(v)
		} else {
			var has bool
			if strings.HasPrefix(v, "uuid.") {
				v = "_udf." + v
			}
			value, has = state.GetValue(v)
			if !has {
				return fmt.Errorf("failed to autogenerate value for %v - unable to eval: %v", k, v)
			}
		}
		state.SetValue(k, value)
	}
	return nil
}

// PostIncrementIfNeeded increments all specified counters by one.
func (d *TableData) PostIncrementIfNeeded(state data.Map) {
	for _, key := range d.PostIncrement {
		keyText := toolbox.AsString(key)
		value, has := state.GetValue(keyText)
		if !has {
			value = 0
		}
		state.SetValue(keyText, toolbox.AsInt(value)+1)
	}
}

// GetValues a table records.
func (d *TableData) GetValues(state data.Map) []map[string]interface{} {
	if d.Data == nil { //backward compatible check
		d.Data = d.Value
	}

	if d.Data == nil {
		return []map[string]interface{}{}
	}
	if toolbox.IsMap(d.Data) {
		var value = d.GetValue(state, d.Data)
		if len(value) == 0 {
			return []map[string]interface{}{}
		}
		return []map[string]interface{}{
			value,
		}
	}
	var result = make([]map[string]interface{}, 0)
	if toolbox.IsSlice(d.Data) {
		var aSlice = toolbox.AsSlice(d.Data)
		for _, item := range aSlice {
			value := d.GetValue(state, item)
			if len(value) > 0 {
				result = append(result, value)
			}
		}
	}
	return result
}

func hasNumericKeys(aMap map[string]interface{}) bool {
	for k := range aMap {
		if strings.HasPrefix(k, "$As") {
			return true
		}
	}
	return false
}

// GetValue returns record.
func (d *TableData) GetValue(state data.Map, source interface{}) map[string]interface{} {
	expanded := state.Expand(source)
	value := toolbox.AsMap(expanded)
	//TODO remove this code
	for k, v := range value {
		var textValue = toolbox.AsString(v)
		if strings.HasPrefix(textValue, "$") {
			delete(value, k)
		} else if strings.HasPrefix(textValue, "\\$") {
			value[k] = string(textValue[1:])
		}
	}

	dataStoreState := state.GetMap(ServiceID)
	var key = d.Key
	if key == "" {
		key = d.Table
	}
	if !dataStoreState.Has(key) {
		dataStoreState.Put(key, data.NewCollection())
	}

	records := dataStoreState.GetCollection(key)
	records.Push(value)
	return value
}
