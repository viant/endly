package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model"
)

func TestSendRequestExpandedEmptyRolesRemainJSONArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","info":{"roles":[]}}`))
	}))
	defer server.Close()
	ctx := newContextWithState(nil)
	defer ctx.Close()
	fixture, err := util.NormalizeMap(map[string]interface{}{"identityFixture": map[string]interface{}{"roles": []interface{}{}}}, true)
	require.NoError(t, err)
	state := ctx.State()
	state.SetValue("identityFixture", fixture["identityFixture"])
	expected := map[string]interface{}{"Responses": []interface{}{
		map[string]interface{}{"JSONBody": map[string]interface{}{"info": map[string]interface{}{"roles": "$identityFixture.roles"}}},
	}}
	expanded := state.Expand(expected).(map[string]interface{})
	expanded, err = util.NormalizeMap(expanded, true)
	require.NoError(t, err)
	roles := expanded["Responses"].([]interface{})[0].(map[string]interface{})["JSONBody"].(map[string]interface{})["info"].(map[string]interface{})["roles"]
	require.Equal(t, []interface{}{}, roles)
	request := &SendRequest{Requests: []*Request{{Method: "GET", URL: server.URL, Repeater: &model.Repeater{Repeat: 1}}}, Expect: expanded}
	require.NoError(t, request.Init())
	response, err := newServiceForTest().send(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, response.Assert)
	require.Empty(t, response.Assert.Failures)
}
