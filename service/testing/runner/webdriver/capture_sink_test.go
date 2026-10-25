package webdriver

import (
	"encoding/json"
	"fmt"
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

func TestCaptureSinkRetainsBoundedResponsesWithoutDuplicateWrites(t *testing.T) {
	file := t.TempDir() + "/capture.jsonl"
	state := newCaptureState(&CaptureStartRequest{MaxEntries: 2})
	if err := state.StartSink(afs.New(), "file://"+file, 1); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		state.mux.Lock()
		state.finishLocked(fmt.Sprint(i), &NetworkTransaction{RequestID: fmt.Sprint(i), URL: "https://example.test/rpc", Status: 200})
		state.mux.Unlock()
		if err := state.FlushSink(); err != nil {
			t.Fatal(err)
		}
		if err := state.FlushSink(); err != nil {
			t.Fatal(err)
		}
	}
	_, retained := state.Snapshot(10, false, true)
	if len(retained) != 2 || retained[0].Sequence != 4 || retained[1].Sequence != 5 {
		t.Fatalf("unexpected retained response history: %+v", retained)
	}
	if err := state.CloseSink(); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if len(lines) != 5 {
		t.Fatalf("sink wrote %d records, want 5", len(lines))
	}
	for i, line := range lines {
		var event struct{ Tx NetworkTransaction }
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event.Tx.Sequence != i+1 {
			t.Fatalf("duplicate/missing sink response at %d", i)
		}
	}
}
