package dsunit

import (
	"fmt"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

//AsTableRecords converts data spcified by dataKey into slice of *TableData to create dsunit data as map[string][]map[string]interface{} (table with records)
func AsTableRecords(dataKey interface{}, state data.Map) (interface{}, error) {
	var recordsKey = fmt.Sprintf("%v.tableRecord", dataKey)
	var result = make(map[string][]map[string]interface{})
	if state == nil {
		return nil, fmt.Errorf("state was nil")
	}
	source, has := state.GetValue(toolbox.AsString(dataKey))

	if !has || source == nil {
		if result, ok := dataKey.(*data.Collection); ok {
			source = result
		} else {
			return nil, fmt.Errorf("value for specified key was empty: %v", dataKey)
		}
	}

	if state.Has(recordsKey) {
		var records = state.Get(recordsKey)
		if result, ok := records.(map[string][]map[string]interface{}); ok {
			return result, nil
		}
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
		Value:   dataStoreState.AsEncodableMap(),
	}
	err := variable.PersistValue()
	if err != nil {
		return nil, err
	}
	state.Put(recordsKey, result)
	return result, nil
}
