package webdriver

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/data/udf"
)

type cdpResultOwner struct {
	raw json.RawMessage
	err error
}

func (o *cdpResultOwner) CDPCommand(string, map[string]interface{}) (json.RawMessage, error) {
	return o.raw, o.err
}
func (o *cdpResultOwner) OtherMethod() (json.RawMessage, error) { return o.raw, o.err }

func TestCDPResultNativeCookieSelection(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		count      int
	}{
		{"zero", `{"cookies":[]}`, 0},
		{"one", `{"cookies":[{"name":"studio_session","value":"synthetic"}]}`, 1},
		{"many", `{"cookies":[{"name":"other","value":"x"},{"name":"studio_session","value":"synthetic"}]}`, 1},
		{"missing", `{}`, 0},
		{"null", `{"cookies":null}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := &cdpResultOwner{raw: json.RawMessage(tc.body)}
			response := &ServiceCallResponse{}
			require.NoError(t, (&service{}).callMethod(owner, "CDPCommand", response, []interface{}{"Network.getCookies", map[string]interface{}{}}))
			require.Len(t, response.Result, 2)
			require.Nil(t, response.Result[1])
			state := data.Map{"cookies": response}
			selected, err := udf.Select("cookies.Result[0].cookies[name = studio_session]", state)
			require.NoError(t, err)
			require.Len(t, selected, tc.count)
			require.Equal(t, tc.body, string(owner.raw), "the response decoder does not mutate producer bytes")
		})
	}
}

func TestCDPResultPreservesLiteralNestedValues(t *testing.T) {
	owner := &cdpResultOwner{raw: json.RawMessage(`{"null":null,"false":false,"zero":0,"empty":[],"array":[false,0,null],"cookies":[{"name":"studio_session","value":"$NeverExecute()"}]}`)}
	response := &ServiceCallResponse{}
	require.NoError(t, (&service{}).callMethod(owner, "CDPCommand", response, []interface{}{"Network.getCookies", map[string]interface{}{}}))
	state := data.Map{"cookies": response}
	for _, tc := range []struct {
		path string
		want interface{}
	}{
		{"cookies.Result[0].null", nil}, {"cookies.Result[0].false", false},
		{"cookies.Result[0].zero", float64(0)}, {"cookies.Result[0].empty", []interface{}{}},
		{"cookies.Result[0].array", []interface{}{false, float64(0), nil}},
	} {
		value, found := state.LookupValue(tc.path)
		require.True(t, found, tc.path)
		require.Equal(t, tc.want, value)
	}
	selected, err := udf.Select("cookies.Result[0].cookies[name = studio_session]", state)
	require.NoError(t, err)
	require.Equal(t, "$NeverExecute()", selected.([]interface{})[0].(map[string]interface{})["value"])
}

func TestCDPResultRejectsMalformedAndNonObjectEnvelopes(t *testing.T) {
	for _, body := range []string{"", "null", "false", "0", "[]", `"private-cookie-canary"`, `{"private-cookie-canary":`} {
		owner := &cdpResultOwner{raw: json.RawMessage(body)}
		err := (&service{}).callMethod(owner, "CDPCommand", &ServiceCallResponse{}, []interface{}{"Network.getCookies", map[string]interface{}{}})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private-cookie-canary")
		require.Less(t, len(err.Error()), 150)
	}
	response := &ServiceCallResponse{}
	owner := &cdpResultOwner{raw: json.RawMessage(`{"cookies":[{"name":"studio_session"},false]}`)}
	require.NoError(t, (&service{}).callMethod(owner, "CDPCommand", response, []interface{}{"Network.getCookies", map[string]interface{}{}}))
	value, err := udf.Select("cookies.Result[0].cookies[name = studio_session]", data.Map{"cookies": response})
	require.Error(t, err)
	require.Empty(t, value, "a malformed collection must not return partial matches")
}

func TestCDPResultPreservesMethodErrorsAndOtherTypes(t *testing.T) {
	sentinel := errors.New("synthetic method failure")
	owner := &cdpResultOwner{raw: json.RawMessage(`not JSON`), err: sentinel}
	response := &ServiceCallResponse{}
	require.ErrorIs(t, (&service{}).callMethod(owner, "CDPCommand", response, []interface{}{"Network.getCookies", map[string]interface{}{}}), sentinel)
	require.Same(t, sentinel, response.Result[1])
	owner.err = nil
	require.NoError(t, (&service{}).callMethod(owner, "OtherMethod", response, nil))
	require.IsType(t, json.RawMessage{}, response.Result[0])
	require.Equal(t, owner.raw, response.Result[0])
	require.Nil(t, response.Result[1])
}
