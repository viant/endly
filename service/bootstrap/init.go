package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viant/afsc/gs"
	"github.com/viant/scy/auth/gcp"
	"github.com/viant/scy/auth/gcp/client"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const (
	gcpCredsEnv       = "VIANT_E2E_GCP_CREDS"
	googleAppCredsEnv = "GOOGLE_APPLICATION_CREDENTIALS"
	defaultCredRel      = ".secret/viant-e2e.json"
)

func init() {
	if data, err := credentialsJSON(); err != nil {
		panic(err)
	} else if data != nil {
		gs.SetOptions(option.WithCredentialsJSON(data))
		return
	}
	srv := gcp.New(client.NewGCloud())
	gs.SetOptions(option.WithTokenSource(&tokenSource{Service: srv}))
}

func credentialsJSON() ([]byte, error) {
	if raw := strings.TrimSpace(os.Getenv(gcpCredsEnv)); raw != "" {
		if !json.Valid([]byte(raw)) {
			return nil, fmt.Errorf("%s: invalid JSON", gcpCredsEnv)
		}
		return []byte(raw), nil
	}
	if path := strings.TrimSpace(os.Getenv(googleAppCredsEnv)); path != "" {
		return readCredFile(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	defaultPath := filepath.Join(home, defaultCredRel)
	if _, err := os.Stat(defaultPath); err != nil {
		return nil, nil
	}
	return readCredFile(defaultPath)
}

func readCredFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials %s: %w", path, err)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("%s: invalid JSON", path)
	}
	return data, nil
}

type tokenSource struct {
	*gcp.Service
}

func (s *tokenSource) Token() (*oauth2.Token, error) {
	gcpScopes := append(gcp.Scopes, "https://www.googleapis.com/auth/bigquery")
	token, err := s.Auth(context.Background(), gcpScopes...)
	if err != nil {
		return nil, err
	}
	return &token.Token, nil
}
