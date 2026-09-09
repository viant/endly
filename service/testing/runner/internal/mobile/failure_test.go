package mobile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/viant/afs"
)

func TestCaptureAppiumFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := interface{}(map[string]interface{}{"sessionId": "session-1"})
		switch r.URL.Path {
		case "/session/session-1/screenshot":
			value = base64.StdEncoding.EncodeToString([]byte("png"))
		case "/session/session-1/source":
			value = "<root>secret</root>"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
	}))
	defer server.Close()
	client, _ := NewAppiumClient(server.URL, server.Client())
	session, err := client.NewSession(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	logPath := directory + "/active.log"
	videoPath := directory + "/segment.mp4"
	if err := os.WriteFile(logPath, []byte("old-new-log"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	failure := CaptureAppiumFailure(context.Background(), afs.New(), session, "failed-step", "assertion failed", &FailureArtifactOptions{
		Directory: directory, MaxSourceBytes: 6,
		Files: []FailureArtifactFile{
			{Path: logPath, Kind: "logcat", MaxBytes: 7, Tail: true},
			{Path: videoPath, Kind: "video", MaxBytes: 100},
		},
	})
	if failure == nil || len(failure.Artifacts) != 5 || len(failure.Errors) != 0 {
		t.Fatalf("unexpected failure evidence: %+v", failure)
	}
	for _, artifact := range failure.Artifacts {
		if _, err := os.Stat(artifact.URL); err != nil {
			t.Fatalf("artifact %s was not written: %v", artifact.URL, err)
		}
	}
}
