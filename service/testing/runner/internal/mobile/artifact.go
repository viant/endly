package mobile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/viant/afs"
	afsurl "github.com/viant/afs/url"
)

type Evidence struct {
	Kind      string
	URL       string
	Size      int
	Sensitive bool
}

func WriteEvidence(ctx context.Context, fs afs.Service, directory, name, kind string, data []byte, sensitive bool) (*Evidence, error) {
	if fs == nil {
		fs = afs.New()
	}
	if strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("artifact directory is required")
	}
	if err := validateEvidenceName(name); err != nil {
		return nil, err
	}
	URL := afsurl.Join(directory, name)
	if err := fs.Upload(ctx, URL, os.FileMode(0o600), bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("write %s evidence: %w", kind, err)
	}
	return &Evidence{Kind: kind, URL: URL, Size: len(data), Sensitive: sensitive}, nil
}

// CopyEvidenceFile streams a stable local capture into an AFS artifact. A tail
// copy is useful for bounded logs; non-tail files are rejected rather than
// truncated when they exceed maxBytes so videos remain valid containers.
func CopyEvidenceFile(ctx context.Context, fs afs.Service, directory, name, kind, sourcePath string, maxBytes int64, tail, sensitive bool) (*Evidence, error) {
	if fs == nil {
		fs = afs.New()
	}
	if strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("artifact directory is required")
	}
	if err := validateEvidenceName(name); err != nil {
		return nil, err
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open %s evidence source: %w", kind, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s evidence source: %w", kind, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s evidence source %q is not a regular file", kind, sourcePath)
	}
	size := info.Size()
	if maxBytes > 0 && size > maxBytes {
		if !tail {
			return nil, fmt.Errorf("%s evidence source is %d bytes, exceeding limit %d", kind, size, maxBytes)
		}
		if _, err := file.Seek(size-maxBytes, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek %s evidence tail: %w", kind, err)
		}
		size = maxBytes
	}
	URL := afsurl.Join(directory, name)
	if err := fs.Upload(ctx, URL, os.FileMode(0o600), io.LimitReader(file, size)); err != nil {
		return nil, fmt.Errorf("copy %s evidence: %w", kind, err)
	}
	return &Evidence{Kind: kind, URL: URL, Size: int(size), Sensitive: sensitive}, nil
}

func validateEvidenceName(name string) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || name == "." || name == ".." {
		return fmt.Errorf("unsafe artifact name %q", name)
	}
	return nil
}
