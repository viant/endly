package mobile

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type captureRunner struct{ stopped atomic.Bool }

func (c *captureRunner) Run(context.Context, Command) (Result, error) { return Result{}, nil }

func (c *captureRunner) Start(_ context.Context, _ Command, stdout, _ io.Writer) (*Process, error) {
	_, _ = stdout.Write([]byte("captured\n"))
	done := make(chan error)
	return &Process{PID: 99, Done: done, Stop: func(context.Context) error {
		c.stopped.Store(true)
		close(done)
		return nil
	}}, nil
}

func TestLoggedProcessLifecycle(t *testing.T) {
	runner := &captureRunner{}
	path := filepath.Join(t.TempDir(), "capture.log")
	process, err := StartLoggedProcess(context.Background(), runner, Command{Name: "capture"}, path)
	if err != nil {
		t.Fatal(err)
	}
	if process.PID != 99 {
		t.Fatalf("unexpected PID: %d", process.PID)
	}
	if err := process.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !runner.stopped.Load() {
		t.Fatal("capture process was not stopped")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "captured\n" {
		t.Fatalf("unexpected capture %q, err=%v", data, err)
	}
}
