package webdriver

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/endly"
)

type boundaryCaptureDriver struct {
	selenium.WebDriver
	pending map[selog.Type][]selog.Message
	failure error
}

func (d *boundaryCaptureDriver) SessionID() string { return "boundary" }
func (d *boundaryCaptureDriver) Quit() error       { return nil }
func (d *boundaryCaptureDriver) Log(kind selog.Type) ([]selog.Message, error) {
	if d.failure != nil {
		return nil, d.failure
	}
	result := d.pending[kind]
	delete(d.pending, kind)
	return result, nil
}
func (d *boundaryCaptureDriver) CDPCommand(_ string, _ map[string]interface{}) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
func TestCaptureStartDiscardsEarlierCDPEvents(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &boundaryCaptureDriver{pending: map[selog.Type][]selog.Message{
		selog.Performance: {{Message: `{"message":{"method":"Network.requestWillBeSent","params":{"requestId":"old","request":{"url":"http://127.0.0.1/mcp","method":"POST","postData":"old"}}}}`}},
		selog.Browser:     {{Message: "earlier console"}},
	}}
	sess := &Session{SessionID: "boundary", driver: driver, Backend: "cdp", Browser: ChromeBrowser}
	putSession(ctx, "boundary", sess)
	yes := true
	_, err := (&service{}).captureStart(ctx, &CaptureStartRequest{SessionID: "boundary", IncludeBodies: &yes})
	require.NoError(t, err)
	require.Empty(t, driver.pending)
	driver.pending[selog.Performance] = []selog.Message{
		{Message: `{"message":{"method":"Network.requestWillBeSent","params":{"requestId":"new","request":{"url":"http://127.0.0.1/mcp","method":"POST","postData":"new"}}}}`},
		{Message: `{"message":{"method":"Network.loadingFinished","params":{"requestId":"new","timestamp":2}}}`},
	}
	sess.Capture.DrainNow(sess)
	_, network := sess.Capture.Snapshot(10, false, true)
	require.Len(t, network, 1)
	require.Equal(t, "new", network[0].RequestID)
	require.Equal(t, "new", network[0].RequestBody.Data)
}
func TestCaptureStartRejectsAnUnverifiableCDPBoundary(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &boundaryCaptureDriver{failure: errors.New("log unavailable")}
	sess := &Session{SessionID: "boundary", driver: driver, Backend: "cdp", Browser: ChromeBrowser}
	putSession(ctx, "boundary", sess)
	_, err := (&service{}).captureStart(ctx, &CaptureStartRequest{SessionID: "boundary"})
	require.ErrorContains(t, err, "capture boundary")
	require.Nil(t, sess.Capture)
}
