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
	failure := CaptureAppiumFailure(context.Background(), afs.New(), session, "failed-step", "assertion failed", &FailureArtifactOptions{
		Directory: directory, MaxSourceBytes: 6,
	})
	if failure == nil || len(failure.Artifacts) != 3 || len(failure.Errors) != 0 {
		t.Fatalf("unexpected failure evidence: %+v", failure)
	}
	for _, artifact := range failure.Artifacts {
		if _, err := os.Stat(artifact.URL); err != nil {
			t.Fatalf("artifact %s was not written: %v", artifact.URL, err)
		}
	}
}
