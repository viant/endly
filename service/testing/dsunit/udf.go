package dsunit

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

// AsTableRecords converts data spcified by dataKey into slice of *TableData to create dsunit data as map[string][]map[string]interface{} (table with records)
func AsTableRecords(dataKey interface{}, state data.Map) (interface{}, error) {
	var outputPrefix string
	if key := toolbox.AsString(dataKey); strings.Count(key, "/") == 1 {
		index := strings.Index(key, "/")
		outputPrefix = key[index+1:]
		dataKey = key[:index]
	}
	var recordsKey = fmt.Sprintf("%v.tableRecord", dataKey)
	var result = make(map[string][]map[string]interface{})
	if state == nil {
		return nil, fmt.Errorf("state was nil")
	}
	source, has := state.GetValue(toolbox.AsString(dataKey))

	if multiTables, ok := source.(data.Map); ok {

		data, err := convertMultiTables(multiTables, state, outputPrefix)
		return data, err
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

const (
	sequencerKey = "dsunitSequencer"
)

func convertMultiTables(tables map[string]interface{}, state data.Map, prefix string) (map[string][]map[string]interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	sequencerMapping := state.GetMap(sequencerKey)
	var sequencer string
	if sequencerMapping == nil {
		sequencerMapping = data.NewMap()
		state.SetValue(sequencerKey, sequencerMapping)
	} else {
		for k := range sequencerMapping {
			if k == "Data" {
				continue
			}
			sequencer = k
			break
		}
	}

	sequencerMappingData := sequencerMapping.GetMap("Data")
	if sequencerMappingData == nil {
		sequencerMappingData = data.NewMap()
		sequencerMapping.SetValue("Data", sequencerMappingData)
	}

	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}

	for table, tableData := range tables {
		if table == "CI_TAXONOMY" {
			fmt.Println(1)
		}
		var tableDataSlice = toolbox.AsSlice(tableData)
		if len(tableDataSlice) == 0 {
			return nil, fmt.Errorf("table %v has no records", table)
		}
		for i, item := range tableDataSlice {
			itemMap, ok := item.(map[string]interface{})
			for k, v := range itemMap {
				if text, ok := v.(string); ok && strings.Contains(text, "${uuid") { //expand uuid
					itemMap[k] = state.ExpandAsText(text)
				}
			}

			if !ok {
				return nil, fmt.Errorf("item %v was not %T", item, itemMap)
			}
			if sequencerKey := allocateTableSequence(itemMap, table, state, sequencerMapping, sequencerMappingData, prefix); sequencerKey != "" {
				sequencer = sequencerKey
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

			caseTag := ""
			caseTagValue, hasTag := itemMap["!tag"]
			if hasTag {
				caseTag = toolbox.AsString(caseTagValue)
				delete(itemMap, "!tag")
			}

			sequencerValues := sequencerMapping.GetMap(sequencer)
			for k, v := range itemMap {
				if v == nil {
					continue
				}
				if _, ok := v.([]string); ok {
					continue
				}
				switch value := v.(type) {
				case string:
					if strings.Contains(value, "$Data") || strings.Contains(value, "$uuid") {
						itemMap[k] = sequencerMapping.Expand(strings.Replace(value, "/", ".", 1))
					} else {
						expanded, ok := expandSequenceExpr(value, toolbox.AsString(caseTag), sequencerMapping, sequencer, sequencerValues, state)
						if ok {
							itemMap[k] = expanded
						}
					}
				case map[interface{}]interface{}:
					for key, keyValue := range value {
						stringKey, ok := key.(string)
						if !ok {
							continue
						}
						expanded, ok := expandSequenceExpr(stringKey, toolbox.AsString(caseTag), sequencerMapping, sequencer, sequencerValues, state)
						if ok {
							value[expanded] = keyValue
							delete(value, key)
						}
					}
				}

			}

			tableDataSlice[i] = itemMap
			result[table] = append(result[table], itemMap)
		}
	}

	return result, nil
}

func expandSequenceExpr(value, caseTag string, sequencerMapping data.Map, sequencer string, sequencerValues data.Map, state data.Map) (interface{}, bool) {
	if caseTag != "" && strings.Contains(value, "${tag}") {
		value = strings.Replace(value, "${tag}", toolbox.AsString(caseTag), 1)
	}

	if strings.Contains(value, "$") {
		expr, key := getSequencerExpr(value, sequencer)

		seqValue, ok := sequencerValues[key]

		if strings.Contains(value, ".parent") {

			fmt.Println("VV: %v %v %v %v %v\n", expr, value, caseTag, seqValue, ok)

		}

		if !ok {
			seqValue, ok = sequencerValues.GetValue(key)
		}
		if !ok {
			return value, false
		}
		value = strings.ReplaceAll(value, expr, toolbox.AsString(seqValue))
		if strings.Contains(value, "$") {
			isInt := strings.HasPrefix(value, "$AsInt")
			if isInt {
				value = value[7 : len(value)-1]
			}
			value = state.ExpandAsText(value)
			if isInt {
				return toolbox.AsInt(value), true
			}
		}
		return value, true
	}
	return value, false
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
	if idx := strings.Index(value, ")"); idx != -1 {
		value = value[:idx]
		if idx = strings.Index(expr, ")"); idx != -1 {
			expr = expr[:idx]
		}
	}
	return expr, value
}

func allocateTableSequence(values map[string]interface{}, table string, state data.Map, tablesMapping, sequenceMappingData data.Map, prefix string) string {
	var seqValue interface{}
	var seqKey string
	var stateKey string
	var sequencer string
	var sequencerKey string

	for k, v := range values {
		if v == nil {
			continue
		}
		value, ok := v.(string)
		if !ok {
			continue
		}
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
			seqValue = actual
			state.SetValue(seqKey, UUID.Version())
		default:

		}
		values[k] = sValue
		key := getKey(value, table)
		if key != "" {
			sequencerKey = seqKey + "/" + key
			stateKey = prefix + table + "." + key
			tablesMapping.SetValue(sequencerKey, seqValue)
		}
	}

	idState := data.NewMap()
	idState.SetValue(seqKey, seqValue)
	for k, v := range values {
		values[k] = idState.Expand(v)
	}
	if stateKey != "" {
		state.SetValue(stateKey, values)
	}

	dataKeySuffix := ""
	if index := strings.LastIndex(sequencerKey, "/"); index != -1 {
		dataKeySuffix = sequencerKey[index+1:]
	}
	dataKey := prefix + table + "." + dataKeySuffix
	sequenceMappingData.SetValue(dataKey, values)

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
