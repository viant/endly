package webdriver

import (
	"fmt"
	"sync"
	"testing"

	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

type quitCountingDriver struct {
	selenium.WebDriver
	quits int
}

func (d *quitCountingDriver) Quit() error {
	d.quits++
	return nil
}

func TestSessionsReturnsSnapshot(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	putSession(ctx, "one", &Session{SessionID: "one"})
	snapshot := Sessions(ctx)
	delete(snapshot, "one")
	if _, ok := lookupSession(ctx, "one"); !ok {
		t.Fatal("mutating snapshot changed session store")
	}
}

func TestSessionStoreConcurrentAccess(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	var wait sync.WaitGroup
	for i := 0; i < 32; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			id := fmt.Sprintf("session-%d", index)
			putSession(ctx, id, &Session{SessionID: id})
			_, _ = lookupSession(ctx, id)
			_ = Sessions(ctx)
		}(i)
	}
	wait.Wait()
	if len(Sessions(ctx)) != 32 {
		t.Fatalf("sessions=%d, wanted 32", len(Sessions(ctx)))
	}
}

func TestSessionCloseIsIdempotent(t *testing.T) {
	driver := &quitCountingDriver{}
	session := &Session{driver: driver}
	session.Close()
	session.Close()
	if driver.quits != 1 {
		t.Fatalf("quits=%d, wanted 1", driver.quits)
	}
}

func TestRunCanDisableAutoStart(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	autoStart := false
	request := &RunRequest{AutoStart: &autoStart, Commands: []interface{}{`page.title()`}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	if _, err := service.run(ctx, request); err == nil {
		t.Fatal("expected missing driver error")
	}
}

func TestSessionCloseDiscardsCaptureAndNetworkObservationState(t *testing.T) {
	driver := &quitCountingDriver{}
	capture := newCaptureState(nil)
	capture.enableConsole = false
	capture.enableNetwork = false
	session := &Session{driver: driver, Capture: capture, Net: &netTracker{}}
	session.Close()
	if capture.Enabled() {
		t.Fatal("closed session left capture enabled")
	}
	if session.Capture != nil || session.Net != nil {
		t.Fatal("closed session retained observation state for the next browser group")
	}
	session.Close()
	if driver.quits != 1 {
		t.Fatalf("quits=%d, wanted 1", driver.quits)
	}
}
