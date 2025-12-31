package webdriver

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/viant/afs"
)

func TestCapture_SinkWritesJSONL(t *testing.T) {
	tmp, err := os.CreateTemp("", "endly-wd-sink-*.jsonl")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)

	includeBodies := false
	state := newCaptureState(&CaptureStartRequest{IncludeBodies: &includeBodies})
	if err := state.StartSink(afs.New(), "file://"+path, 1); err != nil {
		t.Fatalf("StartSink: %v", err)
	}

	state.mux.Lock()
	state.console = append(state.console, &ConsoleEntry{Timestamp: time.Now(), Level: "log", Message: "hello"})
	state.completed = append(state.completed, &NetworkTransaction{RequestID: "1", URL: "https://example.com", Method: "GET", Status: 200})
	state.mux.Unlock()

	if err := state.FlushSink(); err != nil {
		t.Fatalf("FlushSink: %v", err)
	}
	_ = state.CloseSink()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "\"type\":\"console\"") {
		t.Fatalf("expected console event, got: %s", out)
	}
	if !strings.Contains(out, "\"type\":\"network\"") {
		t.Fatalf("expected network event, got: %s", out)
	}
}
