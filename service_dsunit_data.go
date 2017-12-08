package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

//DsUnitTableData represents table data
type DsUnitTableData struct {
	Table         string
	Value         interface{}
	AutoGenerate  map[string]string `json:",omitempty"`
	PostIncrement []string          `json:",omitempty"`
	Key           string
}

//AuotGenerateIfNeeded retrieves auto generated values
func (d *DsUnitTableData) AuotGenerateIfNeeded(state data.Map) error {
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
func (d *DsUnitTableData) PostIncrementIfNeeded(state data.Map) {
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
func (d *DsUnitTableData) GetValues(state data.Map) []map[string]interface{} {
	if toolbox.IsMap(d.Value) {
		return []map[string]interface{}{
			d.GetValue(state, d.Value),
		}
	}
	var result = make([]map[string]interface{}, 0)
	if toolbox.IsSlice(d.Value) {
		var aSlice = toolbox.AsSlice(d.Value)
		for _, item := range aSlice {
			value := d.GetValue(state, item)
			result = append(result, value)
		}
	}
	return result
}

func (d *DsUnitTableData) expandThis(textValue string, value map[string]interface{}) interface{} {
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
func (d *DsUnitTableData) GetValue(state data.Map, source interface{}) map[string]interface{} {
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

	dataStoreState := state.GetMap(DataStoreUnitServiceID)

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

//AsTableRecords converts source into slice of *DsUnitTableData to create dsunit data as map[string][]map[string]interface{} (table with records)
func AsTableRecords(source interface{}, state data.Map) (interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	if source == nil {
		return nil, reportError(fmt.Errorf("Source was nil"))
	}
	if !state.Has(DataStoreUnitServiceID) {
		state.Put(DataStoreUnitServiceID, data.NewMap())
	}

	var prepareTableData, ok = source.([]*DsUnitTableData)

	if ! ok {
		prepareTableData = make([]*DsUnitTableData, 0)
		err := converter.AssignConverted(&prepareTableData, source)
		if err != nil {
			return nil, err
		}
	}
	for _, tableData := range prepareTableData {
		var table = tableData.Table
		err := tableData.AuotGenerateIfNeeded(state)
		if err != nil {
			return nil, err
		}
		result[table] = append(result[table], tableData.GetValues(state)...)
		tableData.PostIncrementIfNeeded(state)
	}
	dataStoreState := state.GetMap(DataStoreUnitServiceID)
	var variable = &Variable{
		Name:    DataStoreUnitServiceID,
		Persist: true,
		Value:   dataStoreState,
	}
	err := variable.PersistValue()
	if err != nil {
		return nil, err
	}
	return result, nil
}
