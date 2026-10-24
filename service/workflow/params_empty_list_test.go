package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRequestInitPreservesEmptyRoleParameters(t *testing.T) {
	request := &RunRequest{URL: "bootstrap-identity", Params: map[string]interface{}{
		"identityFixture": map[string]interface{}{"roles": []interface{}{}},
	}}
	require.NoError(t, request.Init())
	fixture := request.Params["identityFixture"].(map[string]interface{})
	require.Equal(t, []interface{}{}, fixture["roles"])
}
