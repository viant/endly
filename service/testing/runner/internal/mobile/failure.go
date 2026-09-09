package mobile

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/viant/afs"
	"github.com/viant/assertly"
)

type FailureArtifactOptions struct {
	Directory        string
	Screenshot       *bool
	PageSource       *bool
	MaxSourceBytes   int
	Files            []FailureArtifactFile
	CollectionErrors []string
}

type FailureArtifactFile struct {
	Path     string
	Kind     string
	MaxBytes int64
	Tail     bool
}

type FailureEvidence struct {
	Timestamp time.Time
	Reason    string
	Artifacts []*Evidence
	Errors    []string
}

func ValidationFailureReason(validations []*assertly.Validation) (string, bool) {
	reports := []string{}
	for _, validation := range validations {
		if validation != nil && validation.HasFailure() {
			reports = append(reports, validation.Report())
		}
	}
	return strings.Join(reports, "\n"), len(reports) > 0
}

func CaptureAppiumFailure(ctx context.Context, fs afs.Service, session *AppiumSession, prefix, reason string, options *FailureArtifactOptions) *FailureEvidence {
	if options == nil || session == nil || strings.TrimSpace(options.Directory) == "" {
		return nil
	}
	result := &FailureEvidence{
		Timestamp: time.Now().UTC(), Reason: reason, Artifacts: []*Evidence{},
		Errors: append([]string(nil), options.CollectionErrors...),
	}
	if options.Screenshot == nil || *options.Screenshot {
		if data, err := session.Screenshot(ctx); err != nil {
			result.Errors = append(result.Errors, "screenshot: "+err.Error())
		} else if evidence, err := WriteEvidence(ctx, fs, options.Directory, prefix+".png", "screenshot", data, true); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Artifacts = append(result.Artifacts, evidence)
		}
	}
	if options.PageSource == nil || *options.PageSource {
		if source, err := session.PageSource(ctx); err != nil {
			result.Errors = append(result.Errors, "page source: "+err.Error())
		} else {
			maxBytes := options.MaxSourceBytes
			if maxBytes <= 0 {
				maxBytes = 2_000_000
			}
			if len(source) > maxBytes {
				source = source[:maxBytes] + "\n<!-- truncated by Endly -->"
			}
			if evidence, err := WriteEvidence(ctx, fs, options.Directory, prefix+".xml", "pageSource", []byte(source), true); err != nil {
				result.Errors = append(result.Errors, err.Error())
			} else {
				result.Artifacts = append(result.Artifacts, evidence)
			}
		}
	}
	for index, file := range options.Files {
		if strings.TrimSpace(file.Path) == "" || strings.TrimSpace(file.Kind) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("capture file %d: Path and Kind are required", index))
			continue
		}
		extension := filepath.Ext(file.Path)
		if len(extension) > 12 || strings.ContainsAny(extension, `/\\`) {
			extension = ""
		}
		name := fmt.Sprintf("%s-%s-%02d%s", prefix, safeEvidencePart(file.Kind), index+1, extension)
		evidence, err := CopyEvidenceFile(ctx, fs, options.Directory, name, file.Kind, file.Path, file.MaxBytes, file.Tail, true)
		if err != nil {
			result.Errors = append(result.Errors, file.Kind+": "+err.Error())
			continue
		}
		result.Artifacts = append(result.Artifacts, evidence)
	}
	manifest, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		result.Errors = append(result.Errors, "manifest: "+err.Error())
		return result
	}
	evidence, err := WriteEvidence(ctx, fs, options.Directory, prefix+".json", "failureManifest", manifest, true)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("manifest: %v", err))
	} else {
		result.Artifacts = append(result.Artifacts, evidence)
	}
	return result
}

func safeEvidencePart(value string) string {
	result := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
	result = strings.Trim(result, "-")
	if result == "" {
		return "capture"
	}
	return result
}
