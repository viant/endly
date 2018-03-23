package dsunit

import (
	"fmt"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

//TableData represents table data
type TableData struct {
	Table         string
	Value         interface{}
	AutoGenerate  map[string]string `json:",omitempty"`
	PostIncrement []string          `json:",omitempty"`
	Key           string
}

//AutoGenerateIfNeeded retrieves auto generated values
func (d *TableData) AutoGenerateIfNeeded(state data.Map) error {
	for k, v := range d.AutoGenerate {
		value, has := state.GetValue(v)
		if !has {
			return fmt.Errorf("failed to autogenerate value for %v - unable to eval: %v", k, v)
		}
		state.SetValue(k, value)
	}
	return nil
}

//PostIncrementIfNeeded increments all specified counters by one.
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

//GetValues a table records.
func (d *TableData) GetValues(state data.Map) []map[string]interface{} {
	if toolbox.IsMap(d.Value) {
		var value = d.GetValue(state, d.Value)
		if len(value) == 0 {
			return []map[string]interface{}{}
		}
		return []map[string]interface{}{
			value,
		}
	}
	var result = make([]map[string]interface{}, 0)
	if toolbox.IsSlice(d.Value) {
		var aSlice = toolbox.AsSlice(d.Value)
		for _, item := range aSlice {
			value := d.GetValue(state, item)
			if len(value) > 0 {
				result = append(result, value)
			}
		}
	}
	return result
}

func (d *TableData) expandThis(textValue string, value map[string]interface{}) interface{} {
	if strings.Contains(textValue, "this.") {
		var thisState = data.NewMap()
		for subKey, subValue := range value {
			if toolbox.IsString(subValue) {
				subKeyTextValue := toolbox.AsString(subValue)
				if !strings.Contains(subKeyTextValue, "this") {
					thisState.SetValue(fmt.Sprintf("this.%v", subKey), subKeyTextValue)
				}
			}
		}
		return thisState.Expand(textValue)
	}
	return textValue
}

//GetValue returns record.
func (d *TableData) GetValue(state data.Map, source interface{}) map[string]interface{} {
	value := toolbox.AsMap(state.Expand(source))
	for k, v := range value {
		var textValue = toolbox.AsString(v)
		if strings.Contains(textValue, "this") {
			value[k] = d.expandThis(textValue, value)
		} else if strings.HasPrefix(textValue, "$") {
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

//AsTableRecords converts data spcified by dataKey into slice of *TableData to create dsunit data as map[string][]map[string]interface{} (table with records)
func AsTableRecords(dataKey interface{}, state data.Map) (interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	if state == nil {
		return nil, fmt.Errorf("state was nil")
	}

	source, has := state.GetValue(toolbox.AsString(dataKey))
	if !has || source == nil {
		return nil, fmt.Errorf("value for specified key was empty: %v", dataKey)
	}

	if !state.Has(ServiceID) {
		state.Put(ServiceID, data.NewMap())
	}

	var prepareTableData, ok = source.([]*TableData)

	if !ok {
		prepareTableData = make([]*TableData, 0)
		err := converter.AssignConverted(&prepareTableData, source)
		if err != nil {
			return nil, err
		}
	}
	for _, tableData := range prepareTableData {
		var table = tableData.Table
		err := tableData.AutoGenerateIfNeeded(state)
		if err != nil {
			return nil, err
		}
		var values = tableData.GetValues(state)
		if len(values) > 0 {
			result[table] = append(result[table], values...)
			tableData.PostIncrementIfNeeded(state)
		}
	}

	dataStoreState := state.GetMap(ServiceID)
	var variable = &model.Variable{
		Name:    ServiceID,
		Persist: true,
		Value:   dataStoreState,
	}
	err := variable.PersistValue()
	if err != nil {
		return nil, err
	}
	return result, nil
}
