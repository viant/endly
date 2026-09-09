package mobile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/afs"
)

func TestWriteEvidence(t *testing.T) {
	directory := t.TempDir()
	evidence, err := WriteEvidence(context.Background(), afs.New(), directory, "screen.png", "screenshot", []byte("png"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.Sensitive || evidence.Size != 3 {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	data, err := os.ReadFile(filepath.Join(directory, "screen.png"))
	if err != nil || string(data) != "png" {
		t.Fatalf("unexpected artifact %q, err=%v", data, err)
	}
	if _, err := WriteEvidence(context.Background(), afs.New(), directory, "../escape", "source", nil, true); err == nil {
		t.Fatal("expected unsafe name to fail")
	}
}

func TestCopyEvidenceFileTailsLogsAndRejectsTruncatedVideo(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(t.TempDir(), "capture.log")
	if err := os.WriteFile(source, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence, err := CopyEvidenceFile(context.Background(), afs.New(), directory, "failure.log", "log", source, 4, true, true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(evidence.URL)
	if err != nil || string(data) != "6789" || evidence.Size != 4 {
		t.Fatalf("unexpected tailed evidence=%q metadata=%+v err=%v", data, evidence, err)
	}
	if _, err := CopyEvidenceFile(context.Background(), afs.New(), directory, "failure.mp4", "video", source, 4, false, true); err == nil {
		t.Fatal("expected oversized video to be rejected instead of truncated")
	}
}
