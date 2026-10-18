package model

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestVariableUDFContainerExpansionDetachedAndBounded(t *testing.T) {
	canonical := map[string]interface{}{"id": "$current", "literal": "ordinary text", "missing": "$missing", "cycle": "$cycle", "bounded": "${first}"}
	list := []interface{}{canonical}
	state := data.Map{"current": 7, "cycle": "$cycle", "first": "$second", "second": 77}
	state["Object"] = func(interface{}, data.Map) (interface{}, error) { return canonical, nil }
	state["List"] = func(interface{}, data.Map) (interface{}, error) { return list, nil }
	for _, source := range []string{"$Object()", "$List()"} {
		variable := NewVariable("result", "", "", false, source, nil, nil, false)
		for _, id := range []int{7, 8} {
			state["current"] = id
			require.NoError(t, variable.Apply(state, state))
			var object map[string]interface{}
			if source == "$Object()" {
				object = state["result"].(map[string]interface{})
			} else {
				object = state["result"].([]interface{})[0].(map[string]interface{})
			}
			require.Equal(t, id, object["id"])
			require.Equal(t, "ordinary text", object["literal"])
			require.Equal(t, "$missing", object["missing"])
			require.Equal(t, "$cycle", object["cycle"])
			require.Equal(t, "$second", object["bounded"], "returned leaves are expanded once")
			object["id"] = 999
			require.Equal(t, "$current", canonical["id"], "canonical UDF source must remain unchanged")
		}
	}
}

func TestVariableUDFResultMatchesNativeVariableConstruction(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value func() interface{}
	}{
		{"object", func() interface{} {
			return map[string]interface{}{"empty": []interface{}{}, "id": "$current", "literal": "text"}
		}},
		{"list", func() interface{} {
			return []interface{}{map[string]interface{}{"empty": []interface{}{}, "id": "$current"}}
		}},
		{"scalar", func() interface{} { return 17 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			canonical := tc.value()
			state := data.Map{"current": 7}
			state["Produced"] = func(interface{}, data.Map) (interface{}, error) { return canonical, nil }
			deferred := NewVariable("deferred", "", "", false, "$Produced()", nil, nil, false)
			native, err := newVariableFromKeyValuePair("native", tc.value())
			require.NoError(t, err)
			require.NoError(t, deferred.Apply(state, state))
			require.NoError(t, native.Apply(state, state))
			require.Equal(t, state["native"], state["deferred"])
			if object, ok := canonical.(map[string]interface{}); ok {
				require.IsType(t, []interface{}{}, object["empty"], "normalization must not rewrite canonical UDF result")
			}
		})
	}
}

func TestVariableUDFContainerDetachmentDoesNotEvaluateLeaves(t *testing.T) {
	calls := 0
	canonical := map[string]interface{}{"value": "$Count()", "literal": "ordinary text", "missing": "$NotRegistered()"}
	state := data.Map{"invocation": 11}
	state["Produced"] = func(interface{}, data.Map) (interface{}, error) { return canonical, nil }
	state["Count"] = func(_ interface{}, current data.Map) (interface{}, error) { calls++; return current["invocation"], nil }
	variable := NewVariable("result", "", "", false, "$Produced()", nil, nil, false)
	for _, id := range []int{11, 22} {
		state["invocation"] = id
		before := calls
		require.NoError(t, variable.Apply(state, state))
		require.Equal(t, before+1, calls, "leaf UDF must execute only once, using invocation state")
		result := state["result"].(map[string]interface{})
		require.Equal(t, id, result["value"])
		require.Equal(t, "ordinary text", result["literal"])
		require.Equal(t, "$NotRegistered()", result["missing"])
		require.Equal(t, "$Count()", canonical["value"], "detachment and expansion must preserve canonical source")
	}
}
