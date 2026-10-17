package webdriver

import (
	"testing"
	"time"

	"github.com/tebeka/selenium"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/endly"
)

type finalCaptureDriver struct {
	selenium.WebDriver
	pending []selog.Message
}

func (d *finalCaptureDriver) SessionID() string { return "final-capture" }
func (d *finalCaptureDriver) Quit() error       { return nil }
func (d *finalCaptureDriver) Log(kind selog.Type) ([]selog.Message, error) {
	if kind != selog.Performance {
		return nil, nil
	}
	result := d.pending
	d.pending = nil
	return result, nil
}

func TestCaptureExplicitFlushDoesNotDropCompletionInsideThrottle(t *testing.T) {
	for _, action := range []string{"stop", "export"} {
		t.Run(action, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			defer ctx.Close()
			includeBodies := false
			state := newCaptureState(&CaptureStartRequest{IncludeBodies: &includeBodies})
			state.inflight["read"] = &NetworkTransaction{RequestID: "read", URL: "http://127.0.0.1/api/read", StartTimestamp: 1}
			// A future value makes the throttle deterministic, without sleeps.
			state.lastDrain = time.Now().Add(time.Hour)
			driver := &finalCaptureDriver{pending: []selog.Message{{Message: `{"message":{"method":"Network.loadingFinished","params":{"requestId":"read","timestamp":2,"encodedDataLength":42}}}`}}}
			putSession(ctx, "final-capture", &Session{SessionID: "final-capture", driver: driver, Capture: state})
			svc := &service{}
			var err error
			if action == "stop" {
				_, err = svc.captureStop(ctx, &CaptureStopRequest{SessionID: "final-capture"})
			} else {
				_, err = svc.captureExport(ctx, &CaptureExportRequest{SessionID: "final-capture"})
			}
			if err != nil {
				t.Fatal(err)
			}
			summary := state.Summary()
			if summary.RequestsCompleted != 1 || summary.RequestsInFlight != 0 {
				t.Fatalf("explicit %s discarded completion: %#v", action, summary)
			}
		})
	}
}
