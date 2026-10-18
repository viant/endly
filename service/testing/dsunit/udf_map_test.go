package dsunit_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestAsTableRecordsMultiTableMapRepresentations(t *testing.T) {
	for _, tc := range []struct {
		name string
		wrap func(map[string]interface{}) interface{}
	}{
		{"collection map", func(v map[string]interface{}) interface{} { return data.Map(v) }},
		{"workflow initialized map", func(v map[string]interface{}) interface{} { return v }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := data.NewMap()
			tables := map[string]interface{}{"GENERIC_RECORD": []interface{}{map[string]interface{}{"id": 7, "name": "example"}}}
			state.Put("fixtures", tc.wrap(tables))
			actual, err := dsunit.AsTableRecords("fixtures", state)
			require.NoError(t, err)
			require.Equal(t, map[string][]map[string]interface{}{"GENERIC_RECORD": {{"id": 7, "name": "example"}}}, actual)
		})
	}
}

func TestAsTableRecordsMalformedTablesReturnErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		records interface{}
		want    string
	}{
		{"nil records", nil, "records must be a slice"},
		{"object instead of rows", map[string]interface{}{}, "records must be a slice"},
		{"scalar row", []interface{}{1}, "record 0 must be a map"},
		{"nil row", []interface{}{nil}, "record 0 must be a map"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := data.NewMap()
			state.Put("fixtures", map[string]interface{}{"GENERIC_RECORD": tc.records})
			result, err := dsunit.AsTableRecords("fixtures", state)
			require.ErrorContains(t, err, tc.want)
			require.Nil(t, result)
		})
	}
}
