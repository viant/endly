package mobile

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestREPLExecutesCommandsAndInspectorActions(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		":status",
		":tree Increment",
		`value = app.getByTestId("count").text()`,
		"!3",
		":screenshot",
		":history",
		":quit",
	}, "\n"))
	output := &bytes.Buffer{}
	result, err := RunREPL(context.Background(), input, output, REPLConfig{Prompt: "ios> "}, REPLCallbacks{
		Execute: func(_ context.Context, _ string) (*ExecutionResult, error) {
			return &ExecutionResult{Data: map[string]interface{}{"value": "Count: 1"}, Steps: []ExecutionStep{{Attempts: 1, DurationMs: 2}}}, nil
		},
		Source: func(context.Context) (string, error) {
			return `<AppiumAUT><XCUIElementTypeButton name="increment" label="Increment"/></AppiumAUT>`, nil
		},
		Screenshot: func(context.Context) (*Evidence, error) {
			return &Evidence{Kind: "screenshot", URL: "/tmp/screen.png", Size: 42}, nil
		},
		Status: func(context.Context) interface{} { return map[string]interface{}{"session": "ios-1"} },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Commands != 2 || result.Errors != 0 || result.ExitedBy != "quit" || len(result.Artifacts) != 1 {
		t.Fatalf("unexpected REPL result: %+v", result)
	}
	for _, expected := range []string{"Mobile REPL", `"session": "ios-1"`, `label="Increment"`, "value = Count: 1", "/tmp/screen.png", "3  value ="} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %q in output:\n%s", expected, output.String())
		}
	}
}

type synchronizedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(data)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func TestREPLProcessesInputIncrementallyAfterPrompt(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	output := &synchronizedBuffer{}
	executed := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := RunREPL(context.Background(), reader, output, REPLConfig{Prompt: "live> "}, REPLCallbacks{
			Execute: func(_ context.Context, command string) (*ExecutionResult, error) {
				executed <- command
				return &ExecutionResult{Steps: []ExecutionStep{{Attempts: 1}}}, nil
			},
			Source: func(context.Context) (string, error) { return "<root/>", nil },
		})
		done <- err
	}()
	waitForOutput(t, output, "live> ")
	if _, err := io.WriteString(writer, "app.getByText(\"Next\").tap()\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case command := <-executed:
		if command != `app.getByText("Next").tap()` {
			t.Fatalf("unexpected command %q", command)
		}
	case <-time.After(time.Second):
		t.Fatal("REPL did not execute incremental input")
	}
	waitForOutput(t, output, "ok")
	if _, err := io.WriteString(writer, ":quit\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("REPL did not exit")
	}
}

func waitForOutput(t *testing.T, output *synchronizedBuffer, fragment string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(output.String(), fragment) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("output did not contain %q: %s", fragment, output.String())
}

func TestREPLContinuesAfterErrorByDefault(t *testing.T) {
	output := &bytes.Buffer{}
	result, err := RunREPL(context.Background(), strings.NewReader("app.bad()\n:quit\n"), output, REPLConfig{}, REPLCallbacks{
		Execute: func(context.Context, string) (*ExecutionResult, error) { return nil, errors.New("bad command") },
		Source:  func(context.Context) (string, error) { return "<root/>", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors != 1 || result.ExitedBy != "quit" || !strings.Contains(output.String(), "error: bad command") {
		t.Fatalf("unexpected REPL result=%+v output=%s", result, output.String())
	}
}

func TestREPLCloseClosesSessionAndExits(t *testing.T) {
	closed := false
	result, err := RunREPL(context.Background(), strings.NewReader(":close\n"), &bytes.Buffer{}, REPLConfig{}, REPLCallbacks{
		Execute: func(context.Context, string) (*ExecutionResult, error) { return &ExecutionResult{}, nil },
		Source:  func(context.Context) (string, error) { return "<root/>", nil },
		Close:   func(context.Context) error { closed = true; return nil },
	})
	if err != nil || !closed || result.ExitedBy != "close" {
		t.Fatalf("close failed: result=%+v closed=%t err=%v", result, closed, err)
	}
}
