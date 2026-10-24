package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
)

func TestNormalizeMapKeepsEmptyListsInNestedWorkflowInputs(t *testing.T) {
	source := map[string]interface{}{
		"identityFixture": map[string]interface{}{"roles": []interface{}{}},
		"request": map[string]interface{}{
			"Expect": map[string]interface{}{
				"JSONBody": map[string]interface{}{"info": map[string]interface{}{"roles": "$identityFixture.roles"}},
			},
		},
	}
	normalized, err := NormalizeMap(source, true)
	require.NoError(t, err)
	fixture := normalized["identityFixture"].(map[string]interface{})
	require.Equal(t, []interface{}{}, fixture["roles"])
	state := data.Map(normalized)
	expanded := state.Expand(normalized["request"]).(map[string]interface{})
	roles := expanded["Expect"].(map[string]interface{})["JSONBody"].(map[string]interface{})["info"].(map[string]interface{})["roles"]
	require.IsType(t, []interface{}{}, roles)
	require.Empty(t, roles)
	second, err := NormalizeMap(expanded, true)
	require.NoError(t, err)
	require.Equal(t, []interface{}{}, second["Expect"].(map[string]interface{})["JSONBody"].(map[string]interface{})["info"].(map[string]interface{})["roles"])
}

func TestNormalizeMapKeepsNonemptyListsAndConvertsKeyValueMaps(t *testing.T) {
	normalized, err := NormalizeMap(map[string]interface{}{
		"roles":           []interface{}{"ROLE_ADMIN"},
		"nilRoles":        nil,
		"emptyTypedRoles": []string{},
		"emptyTypedPairs": []map[string]interface{}{},
		"nested":          map[interface{}]interface{}{"value": 7},
	}, true)
	require.NoError(t, err)
	require.Equal(t, []interface{}{"ROLE_ADMIN"}, normalized["roles"])
	require.Nil(t, normalized["nilRoles"])
	require.Equal(t, []string{}, normalized["emptyTypedRoles"])
	require.Equal(t, []map[string]interface{}{}, normalized["emptyTypedPairs"])
	require.Equal(t, map[string]interface{}{"value": 7}, normalized["nested"])
}
