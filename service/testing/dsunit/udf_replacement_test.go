package dsunit_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestTableRecordCacheRetainsSameSourceAndReplacesNewSource(t *testing.T) {
	state := data.NewMap()
	state.SetValue("allocation.next", 11)
	source := func() []*dsunit.TableData {
		return []*dsunit.TableData{{Table: "EXAMPLE_RECORD", Data: []map[string]interface{}{{"ID": "$allocation.current"}}, AutoGenerate: map[string]string{"allocation.current": "allocation.next"}, PostIncrement: []string{"allocation.next"}}}
	}
	first := source()
	state.SetValue("batch", first)
	converted, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.EqualValues(t, 11, converted.(map[string][]map[string]interface{})["EXAMPLE_RECORD"][0]["ID"])
	require.EqualValues(t, 12, func() interface{} { v, _ := state.GetValue("allocation.next"); return v }())
	same, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.Equal(t, converted, same)
	require.EqualValues(t, 12, func() interface{} { v, _ := state.GetValue("allocation.next"); return v }(), "unchanged fixture conversion must not allocate twice")
	state.SetValue("batch", source())
	replaced, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.EqualValues(t, 12, replaced.(map[string][]map[string]interface{})["EXAMPLE_RECORD"][0]["ID"])
	require.EqualValues(t, 13, func() interface{} { v, _ := state.GetValue("allocation.next"); return v }())
}

func TestTableRecordCacheSeesCollectionAppend(t *testing.T) {
	state := data.NewMap()
	batch := data.NewCollection()
	batch.Push(&dsunit.TableData{Table: "EXAMPLE_RECORD", Data: []map[string]interface{}{{"ID": 11}}})
	state.SetValue("batch", batch)
	first, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.Len(t, first, 1)
	batch.Push(&dsunit.TableData{Table: "NEW_ONLY", Data: []map[string]interface{}{{"ID": 22}}})
	appended, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.Equal(t, map[string][]map[string]interface{}{"EXAMPLE_RECORD": {{"ID": 11}}, "NEW_ONLY": {{"ID": 22}}}, appended)
}

func TestTableRecordCacheDropsUnversionedResult(t *testing.T) {
	state := data.NewMap()
	state.Put("batch.tableRecord", map[string][]map[string]interface{}{"OLD_ONLY": {{"ID": 11}}})
	state.SetValue("batch", []*dsunit.TableData{{Table: "EXAMPLE_RECORD", Data: []map[string]interface{}{{"ID": 22}}}})
	result, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.Equal(t, map[string][]map[string]interface{}{"EXAMPLE_RECORD": {{"ID": 22}}}, result)
}

func TestTableRecordCacheRetainsCollectionAllocation(t *testing.T) {
	state := data.NewMap()
	state.SetValue("allocation.next", 11)
	batch := data.NewCollection()
	batch.Push(&dsunit.TableData{Table: "EXAMPLE_RECORD", Data: []map[string]interface{}{{"ID": "$allocation.current"}}, AutoGenerate: map[string]string{"allocation.current": "allocation.next"}, PostIncrement: []string{"allocation.next"}})
	state.SetValue("batch", batch)
	first, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	second, err := dsunit.AsTableRecords("batch", state)
	require.NoError(t, err)
	require.Equal(t, first, second)
	next, _ := state.GetValue("allocation.next")
	require.EqualValues(t, 12, next, "the unchanged collection must retain one-time allocation")
}
