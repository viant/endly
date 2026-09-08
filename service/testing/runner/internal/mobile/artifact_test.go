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
