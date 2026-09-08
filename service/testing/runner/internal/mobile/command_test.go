package mobile

import (
	"context"
	"io"
	"runtime"
	"testing"
	"time"
)

func TestOSRunnerRunPreservesArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses printf")
	}
	result, err := (OSRunner{}).Run(context.Background(), Command{
		Name: "printf",
		Args: []string{"%s", "a value; $(not-a-shell)"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "a value; $(not-a-shell)" {
		t.Fatalf("unexpected stdout %q", result.Stdout)
	}
}

func TestOSRunnerStartStopsOwnedProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX sh")
	}
	process, err := (OSRunner{}).Start(context.Background(), Command{Name: "sh", Args: []string{"-c", "sleep 30"}}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := process.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-process.Done:
	case <-time.After(time.Second):
		t.Fatal("process did not exit")
	}
}

func TestChecksReady(t *testing.T) {
	checks := []Check{{Name: "adb", Status: "missing"}, {Name: "appium", Status: "ok"}}
	if ChecksReady(checks, map[string]bool{"adb": true}) {
		t.Fatal("expected required missing check to fail readiness")
	}
	if !ChecksReady(checks, map[string]bool{"appium": true}) {
		t.Fatal("expected optional missing check not to affect readiness")
	}
}
