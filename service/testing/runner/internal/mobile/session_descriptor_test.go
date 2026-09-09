package mobile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionDescriptorRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "android-session.json")
	written := &SessionDescriptor{
		Platform: "android", SessionID: "friendly", BackendSessionID: "backend-1",
		Endpoint: "http://127.0.0.1:4723", TargetID: "emulator-5554", TestIDStrategy: "resourceId",
	}
	if err := WriteSessionDescriptor(path, written); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("descriptor mode=%o", info.Mode().Perm())
	}
	read, err := ReadSessionDescriptor(path, "Android")
	if err != nil {
		t.Fatal(err)
	}
	if read.BackendSessionID != written.BackendSessionID || read.TestIDStrategy != written.TestIDStrategy || read.Version != SessionDescriptorVersion {
		t.Fatalf("unexpected descriptor: %+v", read)
	}
	if _, err := ReadSessionDescriptor(path, "ios"); err == nil {
		t.Fatal("expected platform mismatch")
	}
	if err := RemoveSessionDescriptor(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("descriptor still exists: %v", err)
	}
}
