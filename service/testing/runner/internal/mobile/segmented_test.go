package mobile

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSegmentedCaptureRotatesAndStops(t *testing.T) {
	started := make(chan int, 10)
	capture, err := StartSegmentedCapture(5*time.Millisecond,
		func(_ context.Context, index int) (*Process, error) {
			started <- index
			done := make(chan error)
			return &Process{PID: index + 1, Done: done, Stop: func(context.Context) error {
				close(done)
				return nil
			}}, nil
		},
		func(_ context.Context, index int) (string, error) {
			return fmt.Sprintf("segment-%d.mp4", index), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	for len(started) < 3 {
		select {
		case <-deadline:
			t.Fatal("segments did not rotate")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	paths, errors := capture.Stop()
	if len(paths) < 3 || len(errors) != 0 {
		t.Fatalf("unexpected segments=%v errors=%v", paths, errors)
	}
	pathsAgain, _ := capture.Stop()
	if len(pathsAgain) != len(paths) {
		t.Fatal("stop was not idempotent")
	}
}
