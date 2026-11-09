package dsunit_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/toolbox/data"
)

func TestAsTableRecordsEmptyTablesAreNoOp(t *testing.T) {
	for _, wrapper := range []struct {
		name string
		wrap func(map[string]interface{}) interface{}
	}{
		{"plain map", func(v map[string]interface{}) interface{} { return v }},
		{"data map", func(v map[string]interface{}) interface{} { return data.Map(v) }},
	} {
		for _, rows := range []struct {
			name  string
			value interface{}
		}{
			{"empty array", []interface{}{}},
			{"typed empty rows", []map[string]interface{}{}},
			{"typed nil rows", []map[string]interface{}(nil)},
		} {
			for _, mixed := range []bool{false, true} {
				name := wrapper.name + "/" + rows.name + "/empty only"
				if mixed {
					name = wrapper.name + "/" + rows.name + "/mixed"
				}
				t.Run(name, func(t *testing.T) {
					state := data.NewMap()
					state.Put("Sequences", data.Map{"EMPTY": 41, "RECORD": 7})
					tables := map[string]interface{}{"EMPTY": rows.value}
					expected := map[string][]map[string]interface{}{}
					if mixed {
						tables["RECORD"] = []interface{}{map[string]interface{}{"ID": "$Sequences.RECORD/first"}}
						expected["RECORD"] = []map[string]interface{}{{"ID": 7}}
					}
					state.Put("fixtures", wrapper.wrap(tables))
					actual, err := dsunit.AsTableRecords("fixtures", state)
					require.NoError(t, err)
					// Omit the table entirely: an empty fixture must not become a
					// deletion/reset instruction or a synthetic empty record.
					require.Equal(t, expected, actual)
					counter, _ := state.GetValue("Sequences.EMPTY")
					require.Equal(t, 41, counter)
				})
			}
		}
	}
}

func TestAsTableRecordsEmptyTableDoesNotHideMalformedTable(t *testing.T) {
	for _, malformed := range []interface{}{nil, "not rows", map[string]interface{}{}, []interface{}{nil}, []interface{}{42}} {
		for _, asDataMap := range []bool{false, true} {
			state := data.NewMap()
			tables := map[string]interface{}{"EMPTY": []interface{}{}, "INVALID": malformed}
			var source interface{} = tables
			if asDataMap {
				source = data.Map(tables)
			}
			state.Put("fixtures", source)
			actual, err := dsunit.AsTableRecords("fixtures", state)
			require.ErrorContains(t, err, "table INVALID")
			require.Nil(t, actual)
		}
	}
}
