package mobile

import (
	"bytes"
	"context"
	"fmt"
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
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || name == "." || name == ".." {
		return nil, fmt.Errorf("unsafe artifact name %q", name)
	}
	URL := afsurl.Join(directory, name)
	if err := fs.Upload(ctx, URL, os.FileMode(0o600), bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("write %s evidence: %w", kind, err)
	}
	return &Evidence{Kind: kind, URL: URL, Size: len(data), Sensitive: sensitive}, nil
}
