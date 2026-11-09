package dsunit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/toolbox/data"
)

func TestAsTableRecordsEmbeddedSequenceAliases(t *testing.T) {
	for _, start := range []int{1, 99, 100, 1001} {
		t.Run(fmt.Sprint(start), func(t *testing.T) {
			state := data.NewMap()
			state.Put("Sequences", data.Map{"RECORD": start, "RELATION": 501})
			state.Put("fixtures", data.Map{
				"RECORD":   []interface{}{map[string]interface{}{"ID": "$Sequences.RECORD/case.1", "STR_ID": "fixture-$Sequences.RECORD/case.1", "CURRENT": "$Sequences.RECORD"}},
				"RELATION": []interface{}{map[string]interface{}{"ID": "$Sequences.RELATION/case.1", "RECORD_ID": "$Sequences.RECORD/case.1", "STR_ID": "relation-$Sequences.RELATION/case.1"}},
			})
			actual, err := dsunit.AsTableRecords("fixtures/owned", state)
			require.NoError(t, err)
			rows := actual.(map[string][]map[string]interface{})
			require.EqualValues(t, start, rows["RECORD"][0]["ID"])
			require.Equal(t, fmt.Sprintf("fixture-%d", start), rows["RECORD"][0]["STR_ID"])
			require.EqualValues(t, start, rows["RECORD"][0]["CURRENT"])
			require.Equal(t, fmt.Sprint(start), fmt.Sprint(rows["RELATION"][0]["RECORD_ID"]))
			require.Equal(t, "relation-501", rows["RELATION"][0]["STR_ID"])
			id, ok := state.GetValue("owned.RECORD.case.1.ID")
			require.True(t, ok)
			require.EqualValues(t, start, id)
		})
	}
}

func TestAsTableRecordsSequenceNamespaceKeepsRowCounter(t *testing.T) {
	state := data.NewMap()
	state.Put("tag", "case")
	state.Put("Sequences", data.Map{"RECORD": 101})
	state.Put("fixtures", data.Map{"RECORD": []interface{}{
		map[string]interface{}{"ID": "$Sequences.RECORD/case.1", "STR_ID": "a$Sequences.RECORD/case"},
		map[string]interface{}{"ID": "$Sequences.RECORD/case.2", "STR_ID": "a$Sequences.RECORD/case"},
	}})
	actual, err := dsunit.AsTableRecords("fixtures/owned", state)
	require.NoError(t, err)
	rows := actual.(map[string][]map[string]interface{})["RECORD"]
	require.Equal(t, "a101/case", rows[0]["STR_ID"])
	require.Equal(t, "a102/case", rows[1]["STR_ID"])
}

func TestAsTableRecordsSequenceLeafAliasesStillResolveBothDirections(t *testing.T) {
	state := data.NewMap()
	state.Put("Sequences", data.Map{"RECORD": 101})
	state.Put("fixtures", data.Map{"RECORD": []interface{}{
		map[string]interface{}{"ID": "$Sequences.RECORD/case.1", "REFERENCE": "ref-$Sequences.RECORD/case.2"},
		map[string]interface{}{"ID": "$Sequences.RECORD/case.2", "REFERENCE": "ref-$Sequences.RECORD/case.1"},
	}})
	actual, err := dsunit.AsTableRecords("fixtures/owned", state)
	require.NoError(t, err)
	rows := actual.(map[string][]map[string]interface{})["RECORD"]
	require.Equal(t, "ref-102", rows[0]["REFERENCE"])
	require.Equal(t, "ref-101", rows[1]["REFERENCE"])
}

func TestAsTableRecordsSpreadsheetRowStrings(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		t.Run(fmt.Sprint(tagged), func(t *testing.T) {
			state := data.NewMap()
			tag := "spreadsheet_direct_bulkimport"
			state.Put("tag", tag)
			state.Put("Sequences", data.Map{"CI_AD_ORDER": 1})
			var fixtures []interface{}
			for i, suffix := range []string{"", "", "", ".cpsv", ".cpsv_preserve"} {
				rowTag := tag
				if tagged {
					rowTag = "${tag}"
				}
				row := map[string]interface{}{"ID": fmt.Sprintf("$Sequences.CI_AD_ORDER/%s.%d", rowTag, i+1), "STR_ID": "a$Sequences.CI_AD_ORDER/" + rowTag + suffix}
				if tagged {
					row["!tag"] = tag
				}
				fixtures = append(fixtures, row)
			}
			state.Put("fixtures", data.Map{"CI_AD_ORDER": fixtures})
			actual, err := dsunit.AsTableRecords("fixtures/owned", state)
			require.NoError(t, err)
			rows := actual.(map[string][]map[string]interface{})["CI_AD_ORDER"]
			for i, suffix := range []string{"", "", "", ".cpsv", ".cpsv_preserve"} {
				require.EqualValues(t, i+1, rows[i]["ID"])
				require.Equal(t, fmt.Sprintf("a%d/%s%s", i+1, tag, suffix), state.ExpandAsText(rows[i]["STR_ID"].(string)))
			}
		})
	}
}
