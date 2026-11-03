package buildin

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

func TestStateJSONBytesStoredValuesAndPaths(t *testing.T) {
	values := []interface{}{map[string]interface{}{"code": "${known} $unknown", "flag": false, "nil": nil}, []interface{}{0, false, nil, "你好"}, struct {
		ID   int
		Text string
	}{0, "$known"}, json.Number("9007199254740993"), "9007199254740993", 0, false, nil, []byte{0, 255}}
	for _, value := range values {
		state := data.Map{"outer": map[string]interface{}{"value": value}, "known": "must-not-expand"}
		actual, err := StateJSONBytes("outer.value", state)
		require.NoError(t, err)
		expected, err := json.MarshalIndent(value, "", "\t")
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		if toolbox.IsMap(value) || toolbox.IsStruct(value) || toolbox.IsSlice(value) {
			text, err := toolbox.AsIndentJSONText(value)
			require.NoError(t, err)
			require.Equal(t, []byte(text), actual, "structured formatting must match existing AsJSON")
		}
	}
}

type privateMarshalError struct{}

func (privateMarshalError) MarshalJSON() ([]byte, error) {
	return nil, errors.New("private-marshaler-canary")
}

func TestStateJSONBytesRejectsInvalidSourceAndSerialization(t *testing.T) {
	state := data.Map{"counter": 0, "bad": privateMarshalError{}, "function": map[string]interface{}{"value": func() {}}, "channel": make(chan int), "nan": math.NaN()}
	for _, key := range []interface{}{nil, "", " ", false, 0, []string{"bad"}, "missing", "$bad", "counter++", "++counter", "<-counter", "bad", "function", "channel", "nan"} {
		actual, err := StateJSONBytes(key, state)
		require.Error(t, err)
		require.Nil(t, actual)
		require.NotContains(t, err.Error(), "private-marshaler-canary")
	}
	require.Equal(t, 0, state["counter"], "literal serialization must not execute native increment operations")
}
