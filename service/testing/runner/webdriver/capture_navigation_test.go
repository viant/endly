package webdriver

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/endly"
)

type navigationCaptureDriver struct {
	selenium.WebDriver
	pending     []selog.Message
	order       []string
	navigated   bool
	bodyErr     error
	pageTimeout time.Duration
}

func (d *navigationCaptureDriver) SessionID() string { return "navigation-capture" }
func (d *navigationCaptureDriver) Quit() error       { return nil }
func (d *navigationCaptureDriver) SetPageLoadTimeout(timeout time.Duration) error {
	d.pageTimeout = timeout
	return nil
}
func (d *navigationCaptureDriver) Log(kind selog.Type) ([]selog.Message, error) {
	if kind != selog.Performance {
		return nil, nil
	}
	logs := d.pending
	d.pending = nil
	return logs, nil
}
func (d *navigationCaptureDriver) navigate() error {
	d.order = append(d.order, "navigate")
	d.navigated = true
	d.pending = nil // Navigating clears the old CDP response cache and event buffer.
	return nil
}
func (d *navigationCaptureDriver) Get(string) error               { return d.navigate() }
func (d *navigationCaptureDriver) Refresh() error                 { return d.navigate() }
func (d *navigationCaptureDriver) Back() error                    { return d.navigate() }
func (d *navigationCaptureDriver) Forward() error                 { return d.navigate() }
func (d *navigationCaptureDriver) SwitchWindow(string) error      { return d.navigate() }
func (d *navigationCaptureDriver) Close() error                   { return d.navigate() }
func (d *navigationCaptureDriver) OpenTab(string) (string, error) { return "new", d.navigate() }
func (d *navigationCaptureDriver) CurrentURL() (string, error) {
	return "https://fixture.test/next", nil
}
func (d *navigationCaptureDriver) Title() (string, error) { return "Fixture", nil }
func (d *navigationCaptureDriver) CDPCommand(method string, _ map[string]interface{}) (json.RawMessage, error) {
	if method == "Network.getResponseBody" {
		d.order = append(d.order, "body")
		if d.bodyErr != nil {
			return nil, d.bodyErr
		}
		if d.navigated {
			return nil, errors.New("-32000 No resource with given identifier")
		}
		return json.RawMessage(`{"body":"{\"saved\":true}","base64Encoded":false}`), nil
	}
	if method == "Network.getResponseBodyUnexpected" {
		return nil, errors.New("unexpected body command")
	}
	return json.RawMessage(`{}`), d.navigate()
}

func navigationCaptureFixture(t *testing.T) (*endly.Context, *Session, *navigationCaptureDriver) {
	t.Helper()
	ctx := endly.New().NewContext(nil)
	t.Cleanup(func() { ctx.Close() })
	capture := newCaptureState(nil)
	capture.inflight["saved"] = &NetworkTransaction{RequestID: "saved", Method: "POST", URL: "https://fixture.test/write", Status: 200, StartTimestamp: 1}
	// Deterministically reproduce the ordinary Drain's 100ms suppression.
	capture.lastDrain = time.Now().Add(time.Hour)
	driver := &navigationCaptureDriver{pending: []selog.Message{{Message: `{"message":{"method":"Network.loadingFinished","params":{"requestId":"saved","timestamp":2,"encodedDataLength":14}}}`}}}
	session := &Session{SessionID: driver.SessionID(), driver: driver, Capture: capture}
	putSession(ctx, session.SessionID, session)
	return ctx, session, driver
}

func TestNavigationCapturesCompletedBodyBeforeOldDocumentIsDestroyed(t *testing.T) {
	calls := []struct {
		name, method string
		parameters   []interface{}
	}{
		{"guarded Get", "", nil},
		{"Get", "Get", []interface{}{"https://fixture.test/next"}},
		{"Refresh", "Refresh", nil},
		{"Back", "Back", nil},
		{"Forward", "Forward", nil},
		{"SwitchWindow", "SwitchWindow", []interface{}{"new"}},
		{"Close", "Close", nil},
		{"NewTab", "NewTab", []interface{}{"https://fixture.test/next"}},
	}
	for _, command := range []string{"Page.navigate", "Page.reload", "Page.navigateToHistoryEntry", "Target.closeTarget", "Target.activateTarget", "Target.createTarget"} {
		calls = append(calls, struct {
			name, method string
			parameters   []interface{}
		}{command, "CDPCommand", []interface{}{command, map[string]interface{}{}}})
	}
	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			ctx, session, driver := navigationCaptureFixture(t)
			var err error
			if call.method == "" {
				_, err = (&service{}).getWithGuard(ctx, session, "https://fixture.test/next", NavigationOptions{TimeoutMs: 735})
				require.Equal(t, 735*time.Millisecond, driver.pageTimeout, "navigation budget is unchanged")
			} else {
				_, err = (&service{}).callWebDriver(ctx, &WebDriverCallRequest{SessionID: session.SessionID, Call: &MethodCall{Method: call.method, Parameters: call.parameters}})
			}
			require.NoError(t, err)
			require.Equal(t, []string{"body", "navigate"}, driver.order)
			require.Same(t, driver, session.driver, "capture must preserve concrete backend capabilities")
			_, network := session.Capture.Snapshot(10, false, true)
			summary := session.Capture.Summary()
			require.Empty(t, summary.Errors)
			require.Len(t, network, 1)
			require.Equal(t, 200, network[0].Status)
			require.NotNil(t, network[0].ResponseBody)
			require.JSONEq(t, `{"saved":true}`, network[0].ResponseBody.Data)
		})
	}
}

func TestNavigationDrainRetainsBodyFailureDiagnostic(t *testing.T) {
	ctx, session, driver := navigationCaptureFixture(t)
	driver.bodyErr = errors.New("-32000 No resource with given identifier")
	_, err := (&service{}).callWebDriver(ctx, &WebDriverCallRequest{SessionID: session.SessionID, Call: &MethodCall{Method: "Refresh"}})
	require.NoError(t, err, "capture diagnostics do not change navigation behavior")
	require.Equal(t, []string{"body", "navigate"}, driver.order)
	_, network := session.Capture.Snapshot(10, false, true)
	summary := session.Capture.Summary()
	require.Len(t, summary.Errors, 1)
	require.Contains(t, summary.Errors[0], "getResponseBody(saved)")
	require.Contains(t, summary.Errors[0], "-32000")
	require.Len(t, network, 1)
	require.Nil(t, network[0].ResponseBody, "a failed capture must not invent successful body data")
}

type retryNavigationCaptureDriver struct {
	*navigationCaptureDriver
	capture  *CaptureState
	attempts int
}

func (d *retryNavigationCaptureDriver) CDPCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	if method == "Network.getResponseBody" {
		return d.navigationCaptureDriver.CDPCommand(method, params)
	}
	_ = d.navigate()
	d.attempts++
	if d.attempts == 1 {
		// A new response completes between attempts, still inside Drain's throttle.
		d.navigated = false
		d.capture.inflight["second"] = &NetworkTransaction{RequestID: "second", Status: 200}
		d.pending = []selog.Message{{Message: `{"message":{"method":"Network.loadingFinished","params":{"requestId":"second","timestamp":3}}}`}}
		return json.RawMessage(`{"state":"Pending"}`), nil
	}
	return json.RawMessage(`{"state":"Ready"}`), nil
}

func TestNavigationCaptureDrainsBeforeEveryActualRetry(t *testing.T) {
	ctx, session, driver := navigationCaptureFixture(t)
	retry := &retryNavigationCaptureDriver{navigationCaptureDriver: driver, capture: session.Capture}
	session.driver = retry
	_, err := (&service{}).callWebDriver(ctx, &WebDriverCallRequest{SessionID: session.SessionID, Call: &MethodCall{
		Method: "CDPCommand", Parameters: []interface{}{"Page.reload", map[string]interface{}{}},
		Wait: Wait{WaitTimeMs: 500, PollIntervalMs: 1, Expectation: &CallExpectation{Matcher: "equal", Value: map[string]interface{}{"state": "Ready"}}},
	}})
	require.NoError(t, err)
	require.Equal(t, 2, retry.attempts)
	require.Equal(t, []string{"body", "navigate", "body", "navigate"}, driver.order)
	_, network := session.Capture.Snapshot(10, false, true)
	require.Len(t, network, 2)
	for _, transaction := range network {
		require.NotNil(t, transaction.ResponseBody)
	}
	require.Empty(t, session.Capture.Summary().Errors)
}
