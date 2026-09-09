package mobile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistentHistoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	for _, entry := range []string{"one", `app.fill("secret, value")`, "three"} {
		if err := AppendHistory(path, entry); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := LoadHistory(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0] != `app.fill("secret, value")` || entries[1] != "three" {
		t.Fatalf("unexpected history: %v", entries)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("history permission=%v err=%v", info.Mode().Perm(), err)
	}
	if err := ClearHistory(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("history remains: %v", err)
	}
}
