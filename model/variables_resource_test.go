package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/internal/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

func TestResourceVariablesMatchInlineContainers(t *testing.T) {
	root := t.TempDir()
	body := `- name: fixture
  value:
    id: $symbols.demo.ID
    count: 0
    enabled: false
    nullable: null
    emptyList: []
    emptyMap: {}
    literal.key[0]: unchanged
    rows:
    - id: $symbols.demo.ID
      amount: 1.25
      nullable: null
- name: fallback
  when: '1 = 2'
  else:
    nested:
      id: $symbols.demo.ID
      empty: []
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "fixture.yaml"), []byte(body), 0600))
	loaded, err := util.LoadData([]string{root}, "@fixture")
	require.NoError(t, err)
	var previous []byte
	for _, source := range []interface{}{loaded, "@fixture"} {
		variables, err := GetVariables([]string{root}, source)
		require.NoError(t, err)
		state := data.Map{"symbols": map[string]interface{}{"demo": map[string]interface{}{"ID": int64(9007199254740993)}}}
		require.NoError(t, variables.Apply(state, state))
		encoded, err := json.Marshal(state["fixture"])
		require.NoError(t, err)
		require.JSONEq(t, `{"id":9007199254740993,"count":0,"enabled":false,"nullable":null,"emptyList":[],"emptyMap":{},"literal.key[0]":"unchanged","rows":[{"id":9007199254740993,"amount":1.25,"nullable":null}]}`, string(encoded))
		require.IsType(t, int64(0), state["fixture"].(map[string]interface{})["id"])
		fallback, err := json.Marshal(state["fallback"])
		require.NoError(t, err)
		require.JSONEq(t, `{"nested":{"id":9007199254740993,"empty":[]}}`, string(fallback))
		if previous != nil {
			require.Equal(t, previous, encoded)
		}
		previous = encoded
	}
}

func TestResourceVariablesKeepNonListConversionAndLoadErrors(t *testing.T) {
	root := t.TempDir()
	// A scalar that names itself must retain converter behavior, not recurse.
	for _, body := range []string{"'@fixture'", "42", "null", "[]", "name: existing\nvalue: 7\n", "- 'value = 7'\n"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "fixture.yaml"), []byte(body), 0600))
		loaded, err := util.LoadData([]string{root}, "@fixture")
		if err != nil {
			_, actualErr := GetVariables([]string{root}, "@fixture")
			require.EqualError(t, actualErr, err.Error())
			require.IsType(t, err, actualErr)
			continue
		}
		expected := make([]*Variable, 0)
		expectedErr := toolbox.DefaultConverter.AssignConverted(&expected, loaded)
		actual, actualErr := GetVariables([]string{root}, "@fixture")
		if expectedErr != nil {
			require.EqualError(t, actualErr, expectedErr.Error())
		} else {
			require.NoError(t, actualErr)
			require.Equal(t, Variables(expected), actual)
		}
	}
	_, expectedErr := util.LoadData([]string{root}, "@missing")
	_, actualErr := GetVariables([]string{root}, "@missing")
	require.Error(t, expectedErr)
	require.EqualError(t, actualErr, expectedErr.Error())
	require.IsType(t, expectedErr, actualErr)
}
