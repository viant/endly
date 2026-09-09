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

func TestSegmentedCaptureCheckpointFinalizesAndContinues(t *testing.T) {
	started := make(chan int, 4)
	capture, err := StartSegmentedCapture(time.Hour,
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
	if first := <-started; first != 0 {
		t.Fatalf("first segment=%d", first)
	}
	checkpointCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	paths, errors := capture.Checkpoint(checkpointCtx)
	if len(errors) != 0 || len(paths) != 1 || paths[0] != "segment-0.mp4" {
		t.Fatalf("checkpoint paths=%v errors=%v", paths, errors)
	}
	select {
	case next := <-started:
		if next != 1 {
			t.Fatalf("next segment=%d", next)
		}
	case <-time.After(time.Second):
		t.Fatal("capture did not continue after checkpoint")
	}
	paths, errors = capture.Stop()
	if len(errors) != 0 || len(paths) != 2 {
		t.Fatalf("stop paths=%v errors=%v", paths, errors)
	}
}
