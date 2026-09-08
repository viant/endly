package webdriver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viant/afs/url"
	"github.com/viant/endly"
)

var failureSequence atomic.Uint64

type failureMetadata struct {
	Artifact   *FailureArtifact
	Console    []*ConsoleEntry
	Network    []*NetworkTransaction
	Navigation []*NavigationReport
}

func (s *service) captureFailure(context *endly.Context, session *Session, options *FailureArtifactOptions, action *Action, call *MethodCall, runErr error, navigations []*NavigationReport) *FailureArtifact {
	if options == nil || session == nil || session.driver == nil || runErr == nil {
		return nil
	}
	artifact := &FailureArtifact{
		Timestamp:     time.Now().UTC(),
		Error:         runErr.Error(),
		CaptureErrors: make([]string, 0),
	}
	if action != nil && action.Selector != nil {
		artifact.Selector = action.Selector.By + ":" + action.Selector.Value
	}
	if call != nil {
		artifact.Method = call.Method
	}
	artifact.URL, _ = session.driver.CurrentURL()
	artifact.Title, _ = session.driver.Title()
	directory := strings.TrimSpace(context.Expand(options.Directory))
	if directory == "" {
		artifact.CaptureErrors = append(artifact.CaptureErrors, "failure artifact directory was empty")
		return artifact
	}
	base := fmt.Sprintf("failure-%s-%06d", artifact.Timestamp.Format("20060102T150405.000000000Z"), failureSequence.Add(1))

	screenshot := options.Screenshot == nil || *options.Screenshot
	if screenshot {
		data, err := session.driver.Screenshot()
		if err != nil {
			artifact.CaptureErrors = append(artifact.CaptureErrors, "screenshot: "+err.Error())
		} else {
			artifact.ScreenshotURL = url.Join(directory, base+".png")
			if err := s.fs.Upload(context.Background(), artifact.ScreenshotURL, 0o644, bytes.NewReader(data)); err != nil {
				artifact.CaptureErrors = append(artifact.CaptureErrors, "write screenshot: "+err.Error())
				artifact.ScreenshotURL = ""
			}
		}
	}

	pageSource := options.PageSource == nil || *options.PageSource
	if pageSource {
		source, err := session.driver.PageSource()
		if err != nil {
			artifact.CaptureErrors = append(artifact.CaptureErrors, "page source: "+err.Error())
		} else {
			maxBytes := options.MaxSourceBytes
			if maxBytes <= 0 {
				maxBytes = 2_000_000
			}
			if len(source) > maxBytes {
				source = source[:maxBytes] + "\n<!-- truncated by Endly -->"
			}
			artifact.PageSourceURL = url.Join(directory, base+".html")
			if err := s.fs.Upload(context.Background(), artifact.PageSourceURL, 0o644, strings.NewReader(source)); err != nil {
				artifact.CaptureErrors = append(artifact.CaptureErrors, "write page source: "+err.Error())
				artifact.PageSourceURL = ""
			}
		}
	}

	metadata := &failureMetadata{Artifact: artifact, Navigation: navigations}
	if options.IncludeCapture && (session.Capture == nil || !session.Capture.Enabled()) {
		session.Capture = newCaptureState(&CaptureStartRequest{})
		if isChromeLike(session.Browser) {
			if _, err := executeSessionCDP(session, "Network.enable", map[string]any{}); err != nil {
				artifact.CaptureErrors = append(artifact.CaptureErrors, "enable failure network capture: "+err.Error())
			}
			if _, err := executeSessionCDP(session, "Runtime.enable", map[string]any{}); err != nil {
				artifact.CaptureErrors = append(artifact.CaptureErrors, "enable failure console capture: "+err.Error())
			}
		}
	}
	if options.IncludeCapture && session.Capture != nil {
		session.Capture.Drain(session)
		metadata.Console, metadata.Network = session.Capture.Snapshot(500, true, true)
		artifact.Capture = session.Capture.Summary()
	}
	artifact.MetadataURL = url.Join(directory, base+".json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		artifact.CaptureErrors = append(artifact.CaptureErrors, "encode metadata: "+err.Error())
		artifact.MetadataURL = ""
		return artifact
	}
	if err := s.fs.Upload(context.Background(), artifact.MetadataURL, os.FileMode(0o644), bytes.NewReader(data)); err != nil {
		artifact.CaptureErrors = append(artifact.CaptureErrors, "write metadata: "+err.Error())
		artifact.MetadataURL = ""
	}
	return artifact
}
