package dsunit

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/endly/model"
)

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

