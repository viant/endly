package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/endly/service/workflow"
)

func TestNonInteractiveWorkflowReturnsAfterSummary(t *testing.T) {
	// Run the real asynchronous CLI runner: completion must include its summary,
	// without a fixed post-completion grace period for every process invocation.
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	path := filepath.Join(dir, "run.yaml")
	if err = os.WriteFile(path, []byte("pipeline:\n  done:\n    action: nop\n"), 0600); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	runWorkflow(&workflow.RunRequest{URL: path, SummaryFormat: "json"}, false)
	elapsed := time.Since(started)
	data, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatalf("summary not complete on return: %v", err)
	}
	if !json.Valid(data) {
		t.Fatal("incomplete JSON summary on return")
	}
	if elapsed >= 900*time.Millisecond {
		t.Fatalf("completed no-op retained a post-run delay: %v", elapsed)
	}
}
