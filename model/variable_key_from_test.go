package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v3"
)

func TestVariableKeyFromReadsLiteralMapEntries(t *testing.T) {
	type entryKey string
	type entries map[entryKey]interface{}
	want := map[string]interface{}{"$placeholder": "$placeholder"}
	index := entries{"a": "decoy", "A": map[string]interface{}{"B": "decoy"}}
	keys := []string{"0", "1.5", "A.B", "a[1]", "a]b[", "a/b", "$placeholder", "$probe()", "a]tail", ""}
	for _, key := range keys {
		index[entryKey(key)] = want
	}
	index["false"] = false
	state := data.Map{
		"records": &index, "placeholder": "changed",
		"rows": [][]string{{"first"}, {"second"}}, "index": 1,
		data.UDFKey: data.Map{"probe": func(interface{}, data.Map) (interface{}, error) {
			t.Fatal("literal key was evaluated")
			return nil, nil
		}},
	}
	variables, err := GetVariables(nil, []interface{}{
		map[string]interface{}{"name": "selected", "from": "records", "keyFrom": "key", "required": true},
		map[string]interface{}{"name": "indexed", "from": "rows[$index]"},
	})
	require.NoError(t, err)
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			state["key"] = key
			out := data.NewMap()
			require.NoError(t, variables.Apply(state, out))
			require.Equal(t, want, out["selected"])
			require.Equal(t, []string{"second"}, out["indexed"])
			event := NewModifiedStateEvent(variables, state, out)
			require.NotContains(t, event.In, "records")
			require.Equal(t, want, event.Modified["selected"])
		})
	}
	state["key"] = "false"
	out := data.NewMap()
	require.NoError(t, variables.Apply(state, out))
	require.Equal(t, false, out["selected"])

	encoded, err := yaml.Marshal(Variables{variables[0], nil, variables[1]})
	require.NoError(t, err)
	var decoded []interface{}
	require.NoError(t, yaml.Unmarshal(encoded, &decoded))
	restored, err := GetVariables(nil, decoded)
	require.NoError(t, err)
	require.Equal(t, "key", restored[0].KeyFrom)
	require.Equal(t, "records", restored[0].From)
	require.Equal(t, "rows[$index]", restored[1].From)
}

func TestVariableKeyFromEventsRecordOnlyOutputs(t *testing.T) {
	calls := 0
	state := data.Map{
		"records": map[string]interface{}{
			"a": "first", "b": "second", "c": "skipped",
			"d": func() interface{} {
				calls++
				return "called"
			},
		},
		"a":    "a",
		"b":    "b",
		"c":    "c",
		"d":    "d",
		"skip": true,
	}
	variables := Variables{
		{Name: "first", From: "records", KeyFrom: "a"},
		{Name: "second", From: "records", KeyFrom: "b"},
		{Name: "gated", From: "records", KeyFrom: "c", When: "$skip == false"},
		{Name: "getter", From: "records", KeyFrom: "d"},
	}
	out := data.NewMap()
	require.NoError(t, variables.Apply(state, out))
	event := NewModifiedStateEvent(variables, state, out)
	require.Empty(t, event.In)
	require.Equal(t, "first", event.Modified["first"])
	require.Equal(t, "second", event.Modified["second"])
	require.Nil(t, event.Modified["gated"])
	require.Zero(t, calls)
	require.IsType(t, (func() interface{})(nil), event.Modified["getter"])
}

func TestVariableKeyFromMissingAndInvalidSources(t *testing.T) {
	var nilMapPointer *map[string]interface{}
	state := data.Map{"key": "absent"}
	variable := &Variable{Name: "selected", From: "records", KeyFrom: "key", Value: "fallback"}
	out := data.NewMap()
	for _, source := range []interface{}{nil, nilMapPointer, map[string]string(nil), map[string]interface{}{"absent": nil}, map[string]string{"other": "value"}} {
		state["records"] = source
		require.NoError(t, variable.Apply(state, out))
		require.Equal(t, "fallback", out["selected"])
	}
	variable.Value, variable.Required = nil, true
	require.ErrorContains(t, variable.Apply(state, out), "is required")
	state["records"] = map[int]string{0: "value"}
	require.ErrorContains(t, variable.Apply(state, out), "map with string keys")
	state["records"], state["key"] = map[string]string{"0": "value"}, 0
	require.ErrorContains(t, variable.Apply(state, out), "string key")
	state["getter"] = func(string) interface{} {
		t.Fatal("passive key path called a getter")
		return "0"
	}
	variable.KeyFrom = "getter.key"
	require.ErrorContains(t, variable.Apply(state, out), "is required")
	variable.From = ""
	require.ErrorContains(t, variable.Apply(state, out), "requires from")
}

func TestVariableKeyFromNilIntermediatePaths(t *testing.T) {
	var nilMapPointer *map[string]interface{}
	for _, absent := range []interface{}{nilMapPointer, map[string]interface{}(nil)} {
		state := data.Map{
			"oracle":    absent,
			"rowsByKey": map[string]string{"present": "decoy"},
			"context":   absent,
			"key":       "present",
		}
		for _, variable := range []*Variable{
			{Name: "selected", From: "oracle.rowsByKey", KeyFrom: "key", Value: "fallback"},
			{Name: "selected", From: "rowsByKey", KeyFrom: "context.key", Value: "fallback"},
		} {
			out := data.NewMap()
			require.NoError(t, variable.Apply(state, out))
			require.Equal(t, "fallback", out["selected"])
		}
	}
}
