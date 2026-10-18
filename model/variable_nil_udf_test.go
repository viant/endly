package model

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestVariableNilUDFResult(t *testing.T) {
	var nilMap map[string]interface{}
	var nilList []interface{}
	for _, value := range []interface{}{nil, nilMap, nilList} {
		state := data.NewMap()
		state["Produced"] = func(interface{}, data.Map) (interface{}, error) { return value, nil }
		variable := NewVariable("result", "", "", false, "$Produced()", nil, nil, false)
		// Preserve the native expansion result, including its exact typed-nil shape.
		want := state.Expand("$Produced()")
		require.NotPanics(t, func() { require.NoError(t, variable.Apply(state, state)); require.Equal(t, want, state["result"]) })
	}
}
