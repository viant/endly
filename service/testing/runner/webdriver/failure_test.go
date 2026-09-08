package webdriver

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/tebeka/selenium"
	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/endly"
)

type evidenceDriver struct {
	selenium.WebDriver
}

func (d *evidenceDriver) Screenshot() ([]byte, error) { return []byte("png"), nil }
func (d *evidenceDriver) PageSource() (string, error) { return strings.Repeat("x", 32), nil }
func (d *evidenceDriver) CurrentURL() (string, error) { return "https://example.test/failure", nil }
func (d *evidenceDriver) Title() (string, error)      { return "Failure", nil }

func TestCaptureFailure(t *testing.T) {
	manager := endly.New()
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	service := &service{AbstractService: endly.NewAbstractService(ServiceID), fs: afs.New()}
	session := &Session{driver: &evidenceDriver{}}
	artifact := service.captureFailure(ctx, session, &FailureArtifactOptions{
		Directory:      t.TempDir(),
		MaxSourceBytes: 8,
	}, &Action{Selector: NewWebElementSelector("css selector", "#submit")}, &MethodCall{Method: "Click"}, errors.New("boom"), nil)
	if artifact == nil {
		t.Fatal("artifact was nil")
	}
	for _, location := range []string{artifact.ScreenshotURL, artifact.PageSourceURL, artifact.MetadataURL} {
		if location == "" {
			t.Fatalf("artifact location was empty: %#v", artifact)
		}
		if _, err := os.Stat(url.Path(location)); err != nil {
			t.Fatalf("artifact %s: %v", location, err)
		}
	}
	source, err := os.ReadFile(url.Path(artifact.PageSourceURL))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "truncated by Endly") {
		t.Fatalf("expected truncated page source, got %q", source)
	}
}

func TestRunRequestRejectsArtifactOptionsWithoutDirectory(t *testing.T) {
	request := &RunRequest{
		SessionID:        "test",
		Actions:          []*Action{NewAction("", "", "Title")},
		FailureArtifacts: &FailureArtifactOptions{},
	}
	if err := request.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
