package webdriver

import (
	selog "github.com/tebeka/selenium/log"
	"testing"
)

type boundaryLogs struct{ connection directCDPConnection }

func (d *boundaryLogs) Log(kind selog.Type) ([]selog.Message, error) {
	return d.connection.logs(kind), nil
}
func TestCaptureBoundaryDiscardsHistoryAndKeepsNewEvents(t *testing.T) {
	d := &boundaryLogs{}
	d.connection.browser = []selog.Message{{Message: "old console"}}
	d.connection.performance = []selog.Message{{Message: "old metadata read"}}
	discardCaptureBacklog(d)
	d.connection.performance = append(d.connection.performance, selog.Message{Message: "new metadata read"})
	events, _ := d.Log(selog.Performance)
	if len(events) != 1 || events[0].Message != "new metadata read" {
		t.Fatalf("bad capture boundary: %+v", events)
	}
	console, _ := d.Log(selog.Browser)
	if len(console) != 0 {
		t.Fatal("console history leaked")
	}
	discardCaptureBacklog(struct{}{})
}
