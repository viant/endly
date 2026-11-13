package webdriver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

func TestCallDriverCDPUsesChromeDriverTransport(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/session/fake/goog/cdp/execute", r.URL.Path)
		var payload struct {
			Command string         `json:"cmd"`
			Params  map[string]any `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "Emulation.setDeviceMetricsOverride", payload.Command)
		require.Equal(t, float64(390), payload.Params["width"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":{"applied":true}}`))
	}))
	defer server.Close()
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	putSession(ctx, "owned", &Session{SessionID: "owned", Remote: server.URL, driver: &runtimeDriver{}})
	result, err := (&service{}).callWebDriver(ctx, &WebDriverCallRequest{SessionID: "owned", Call: &MethodCall{Method: "CDPCommand", Parameters: []interface{}{"Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 390}}}})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, map[string]interface{}{"applied": true}, result.Data["CDPCommand"])
}

func TestCallDriverCDPViewportLive(t *testing.T) {
	remote := os.Getenv("ENDLY_WEBDRIVER_TEST_URL")
	if remote == "" {
		t.Skip("owned ChromeDriver endpoint required")
	}
	driver, err := selenium.NewRemote(selenium.Capabilities{"browserName": "chrome"}, remote)
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Quit() })
	require.NoError(t, driver.Get("data:text/html,<title>Viewport fixture</title><p>Ready</p>"))
	initial, err := driver.ExecuteScript("return innerWidth;", nil)
	require.NoError(t, err)
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	putSession(ctx, "viewport", &Session{SessionID: "viewport", Remote: remote, driver: driver})
	call := func(command string, params interface{}) {
		_, err := (&service{}).callWebDriver(ctx, &WebDriverCallRequest{SessionID: "viewport", Call: &MethodCall{Method: "CDPCommand", Parameters: []interface{}{command, params}}})
		require.NoError(t, err)
	}
	call("Emulation.setDeviceMetricsOverride", map[string]interface{}{"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": false})
	width, err := driver.ExecuteScript("return innerWidth;", nil)
	require.NoError(t, err)
	require.EqualValues(t, 390, width)
	call("Emulation.clearDeviceMetricsOverride", []interface{}{})
	width, err = driver.ExecuteScript("return innerWidth;", nil)
	require.NoError(t, err)
	require.Equal(t, initial, width)
}
