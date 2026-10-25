package webdriver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type cdpParameterRecorder struct {
	method string
	params map[string]interface{}
}

func (r *cdpParameterRecorder) CDPCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	r.method, r.params = method, params
	return json.RawMessage(`{}`), nil
}

func TestCallMethodCDPEmptyWorkflowObject(t *testing.T) {
	for _, params := range []interface{}{map[string]interface{}{}, []interface{}{}} {
		recorder := &cdpParameterRecorder{}
		original := []interface{}{"Network.clearBrowserCache", params}
		response := &ServiceCallResponse{}
		require.NoError(t, (&service{}).callMethod(recorder, "CDPCommand", response, original))
		require.Equal(t, "Network.clearBrowserCache", recorder.method)
		require.NotNil(t, recorder.params)
		require.Empty(t, recorder.params)
		require.IsType(t, params, original[1], "normalization must not mutate the authored call")
	}
}

func TestCallMethodCDPParameters(t *testing.T) {
	recorder := &cdpParameterRecorder{}
	params := map[string]interface{}{"cacheDisabled": true}
	require.NoError(t, (&service{}).callMethod(recorder, "CDPCommand", &ServiceCallResponse{}, []interface{}{"Network.setCacheDisabled", params}))
	require.Equal(t, params, recorder.params)
	require.Error(t, (&service{}).callMethod(recorder, "CDPCommand", &ServiceCallResponse{}, []interface{}{"Network.clearBrowserCache", []interface{}{1}}))
}
