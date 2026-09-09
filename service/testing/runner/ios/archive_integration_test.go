//go:build darwin

package ios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func TestIOSArchiveIntegration(t *testing.T) {
	if os.Getenv("ENDLY_IOS_ARCHIVE_INTEGRATION") != "1" {
		t.Skip("set ENDLY_IOS_ARCHIVE_INTEGRATION=1 to build a real unsigned xcarchive")
	}
	projectPath := os.Getenv("ENDLY_IOS_TEST_PROJECT")
	if projectPath == "" {
		projectPath = filepath.Join("test", "fixture", "FixtureApp.xcodeproj")
	}
	scheme := os.Getenv("ENDLY_IOS_TEST_SCHEME")
	if scheme == "" {
		scheme = "FixtureApp"
	}
	archivePath := filepath.Join(t.TempDir(), "FixtureApp.xcarchive")
	service := newService(mobile.OSRunner{})
	response, err := service.build(endly.New().NewContext(nil), &BuildRequest{
		ProjectPath: projectPath, Scheme: scheme, Configuration: "Release", Mode: "archive",
		ArchivePath: archivePath, DerivedDataPath: filepath.Join(t.TempDir(), "DerivedData"),
		BuildSettings: map[string]string{"CODE_SIGNING_ALLOWED": "NO"}, TimeoutMs: 10 * 60 * 1000,
	})
	if err != nil {
		if response == nil {
			t.Fatal(err)
		}
		t.Fatalf("archive failed: %v\nstdout:\n%s\nstderr:\n%s", err, response.Stdout, response.Stderr)
	}
	if len(response.Artifacts) != 1 || response.Artifacts[0].Kind != "xcarchive" || response.Artifacts[0].HostPath != archivePath || response.Artifacts[0].Size <= 0 || response.Artifacts[0].SHA256 == "" {
		t.Fatalf("unexpected archive response: %+v", response)
	}
	if info, err := os.Stat(filepath.Join(archivePath, "Info.plist")); err != nil || info.IsDir() {
		t.Fatalf("archive Info.plist missing: %v", err)
	}
}
