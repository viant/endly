package webdriver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
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
	for _, params := range []interface{}{nil, map[string]interface{}{}, []interface{}{}} {
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

func TestRunCDPDecodedEmptyParameters(t *testing.T) {
	for _, params := range []interface{}{nil, []interface{}{}, map[string]interface{}{}} {
		ctx, session, driver := navigationCaptureFixture(t)
		session.Capture = nil
		request := &RunRequest{SessionID: session.SessionID, Actions: []*Action{{Key: "clear", Calls: []*MethodCall{{Method: "CDPCommand", Parameters: []interface{}{"Network.clearBrowserCache", params}}}}}}
		require.NoError(t, request.Init())
		svc := &service{AbstractService: endly.NewAbstractService(ServiceID)}
		_, err := svc.run(ctx, request)
		require.NoError(t, err)
		require.True(t, driver.navigated, "the command must reach the native driver")
	}
}

func TestCallMethodCDPParameters(t *testing.T) {
	recorder := &cdpParameterRecorder{}
	params := map[string]interface{}{"cacheDisabled": true}
	require.NoError(t, (&service{}).callMethod(recorder, "CDPCommand", &ServiceCallResponse{}, []interface{}{"Network.setCacheDisabled", params}))
	require.Equal(t, params, recorder.params)
	require.Error(t, (&service{}).callMethod(recorder, "CDPCommand", &ServiceCallResponse{}, []interface{}{"Network.clearBrowserCache", []interface{}{1}}))
}
