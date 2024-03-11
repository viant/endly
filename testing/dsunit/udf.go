package dsunit

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

// AsTableRecords converts data spcified by dataKey into slice of *TableData to create dsunit data as map[string][]map[string]interface{} (table with records)
func AsTableRecords(dataKey interface{}, state data.Map) (interface{}, error) {
	var recordsKey = fmt.Sprintf("%v.tableRecord", dataKey)
	var result = make(map[string][]map[string]interface{})
	if state == nil {
		return nil, fmt.Errorf("state was nil")
	}

	source, has := state.GetValue(toolbox.AsString(dataKey))
	if multiTables, ok := source.(data.Map); ok {
		return convertMultiTables(multiTables, state)
	}

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
	state.Put(recordsKey, result)
	return result, nil
}

func convertMultiTables(tables map[string]interface{}, state data.Map) (map[string][]map[string]interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	var tablesMapping = data.NewMap()
	var sequencer string

	for table, tableData := range tables {
		var tableDataSlice = toolbox.AsSlice(tableData)
		if len(tableDataSlice) == 0 {
			return nil, fmt.Errorf("table %v has no records", table)
		}

		for i, item := range tableDataSlice {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("item %v was not %T", item)
			}
			if seq := allocateTableSequence(itemMap, table, state, tablesMapping); seq != "" {
				sequencer = seq
			}
			tableDataSlice[i] = itemMap
		}
	}

	for table, tableData := range tables {
		var tableDataSlice = toolbox.AsSlice(tableData)
		if len(tableDataSlice) == 0 {
			return nil, fmt.Errorf("table %v has no records", table)
		}
		for i, item := range tableDataSlice {
			itemMap := item.(map[string]interface{})
			sequencerData := tablesMapping.GetMap(sequencer)
			for k, v := range itemMap {
				value := toolbox.AsString(v)
				if strings.Contains(value, "$") {
					expr, key := getSequencerExpr(value, sequencer)
					seqValue, ok := sequencerData[key]
					if !ok {
						seqValue, ok = sequencerData.GetValue(key)
					}
					if ok {
						itemMap[k] = strings.ReplaceAll(value, expr, toolbox.AsString(seqValue))
					}
				}
			}
			tableDataSlice[i] = itemMap
			result[table] = append(result[table], itemMap)
		}
	}

	return result, nil
}

func getSequencerExpr(value string, sequencer string) (string, string) {
	expr := "$" + sequencer + "."
	index := strings.Index(value, expr)
	if index == -1 {
		expr = "${" + sequencer + "."
		index = strings.Index(value, expr)
	}
	if index == -1 {
		return "", ""
	}
	expr = value[index:]
	value = value[index+1+len(sequencer+"."):]
	return expr, value
}

func allocateTableSequence(values map[string]interface{}, table string, state data.Map, tablesMapping data.Map) string {
	var seqValue int
	var seqKey string
	var stateKey string
	var sequencer string
	for k, v := range values {
		value := toolbox.AsString(v)
		if !hasTableSequence(value, table) {
			continue
		}
		if sequencer == "" {
			sequencer = getSequencer(value, table)
		}
		seqKey = sequencer + "." + table
		sValue, _ := state.GetValue(seqKey)
		switch actual := sValue.(type) {
		case int:
			state.SetValue(seqKey, actual+1)
			seqValue = actual
		case string:
			UUID := uuid.New()
			state.SetValue(seqKey, UUID.String())
		default:

		}
		values[k] = sValue
		key := getKey(value, table)
		if key != "" {
			sequencerKey := seqKey + "/" + key
			stateKey = table + "." + key
			tablesMapping.SetValue(sequencerKey, seqValue)
		}
		break
	}
	idState := data.NewMap()
	idState.SetValue(seqKey, seqValue)
	//seqTextValue := toolbox.AsString(seqValue)
	for k, v := range values {
		value := toolbox.AsString(v)
		values[k] = idState.Expand(value)
	}
	if stateKey != "" {
		state.SetValue(stateKey, values)
	}
	return sequencer
}

func getKey(value string, table string) string {
	if index := strings.LastIndex(value, table+"/"); index != -1 {
		value = value[len(table)+index+1 : len(value)]
		return value
	}
	return ""
}

func getSequencer(value string, table string) string {
	if index := strings.Index(value, "."); index != -1 {
		value = value[1:index]
	}
	return strings.Trim(value, "{}")
}

func hasTableSequence(value string, table string) bool {
	return strings.HasPrefix(value, "$") &&
		(strings.Contains(value, "."+table+"/") ||
			strings.Contains(value, "."+table+"\""))
}
