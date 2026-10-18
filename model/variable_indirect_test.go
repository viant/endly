package model

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestVariableIndirectScalarExpansionIsBounded(t *testing.T) {
	state := data.Map{"first": "$second", "second": "$third", "third": 77, "literal": "$second"}
	variable := NewVariable("result", "", "", false, "${first}", nil, nil, false)
	require.NoError(t, variable.Apply(state, state))
	require.Equal(t, "$third", state["result"], "only one additional expansion is permitted")
	direct := NewVariable("direct", "", "", false, "$first", nil, nil, false)
	require.NoError(t, direct.Apply(state, state))
	require.Equal(t, "$third", state["direct"], "shorthand reference uses the same bounded expansion")
	state["cycle"] = "${cycle}"
	cyclic := NewVariable("result", "", "", false, "${cycle}", nil, nil, false)
	require.NoError(t, cyclic.Apply(state, state))
	require.Equal(t, "${cycle}", state["result"])
}

func TestVariableIndirectScalarLiteralControls(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   interface{}
	}{
		{"ordinary scalar", "ordinary scalar"},
		{"${missing}", "${missing}"},
		{"${left}${right}", "$third"},
		{"${left} text ${right}", "$ text third"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			state := data.Map{"left": "$", "right": "third", "third": 77}
			variable := NewVariable("result", "", "", false, tc.source, nil, nil, false)
			require.NoError(t, variable.Apply(state, state))
			require.Equal(t, tc.want, state["result"])
		})
	}
}
