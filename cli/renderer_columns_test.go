package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/viant/endly/model/msg"
)

type columnFDWriter struct {
	bytes.Buffer
	fd uintptr
}

func BenchmarkRendererColumnsRedirected(b *testing.B) {
	renderer := NewRenderer(io.Discard, 120)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Columns()
	}
}

func BenchmarkCLIRepeatedEventRendering(b *testing.B) {
	listener := NewEventListener(io.Discard)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := msg.NewEvent(msg.NewRepeatedEvent("synthetic step", "workflow.nop"))
		event.SetLoggable(true)
		listener(event)
	}
}

func (w *columnFDWriter) Fd() uintptr { return w.fd }

func TestRendererColumnsRedirectedDoesNotSpawnTerminalProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture; output descriptor contracts are platform independent")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "terminal-probes")
	if err := os.WriteFile(filepath.Join(dir, "stty"), []byte("#!/bin/sh\nprintf 'probe\\n' >> \"$ENDLY_TEST_TERMINAL_PROBES\"\nprintf '24 200\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("ENDLY_TEST_TERMINAL_PROBES", marker)
	renderer := NewRenderer(&bytes.Buffer{}, 120)
	for i := 0; i < 3; i++ {
		if got := renderer.Columns(); got != minColumns {
			t.Errorf("redirected output columns = %d, want fallback %d", got, minColumns)
		}
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("redirected output launched an external terminal probe: %v", err)
	}
}

func TestRendererColumnsUsesOutputDescriptorAndTracksResize(t *testing.T) {
	previous := terminalSize
	t.Cleanup(func() { terminalSize = previous })
	widths := []int{180, 240, 90}
	index := 0
	terminalSize = func(fd int) (int, int, error) {
		if fd != 17 {
			t.Fatalf("terminal descriptor = %d, want output descriptor 17", fd)
		}
		width := widths[index]
		index++
		return width, 30, nil
	}
	renderer := NewRenderer(&columnFDWriter{fd: 17}, 120)
	for _, want := range []int{180, 240, minColumns} {
		if got := renderer.Columns(); got != want {
			t.Errorf("terminal columns = %d, want %d", got, want)
		}
	}
	if index != len(widths) {
		t.Errorf("terminal size queried %d times, want %d to preserve resize", index, len(widths))
	}
	terminalSize = func(int) (int, int, error) { return 0, 0, errors.New("not a terminal") }
	if got := renderer.Columns(); got != minColumns {
		t.Errorf("failed terminal query columns = %d, want %d", got, minColumns)
	}
}
