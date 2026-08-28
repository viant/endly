package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Profile struct {
	Endpoint      string `json:"endpoint"`
	SessionID     string `json:"sessionId,omitempty"`
	JWTPrivateKey string `json:"jwtPrivateKey,omitempty"`
	JWTIssuer     string `json:"jwtIssuer,omitempty"`
	JWTAudience   string `json:"jwtAudience,omitempty"`
	JWTScope      string `json:"jwtScope,omitempty"`
	JWTSubject    string `json:"jwtSubject,omitempty"`
}

type ProfileStore struct {
	Profiles map[string]*Profile `json:"profiles"`
}

func profilePath() string {
	if candidate := os.Getenv("ENDLY_PROFILE_FILE"); candidate != "" {
		return candidate
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".endly", "client.json")
}

func loadProfiles(path string) (*ProfileStore, error) {
	result := &ProfileStore{Profiles: map[string]*Profile{}}
	if path == "" {
		return result, nil
	}
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(payload, result); err != nil {
		return nil, err
	}
	if result.Profiles == nil {
		result.Profiles = map[string]*Profile{}
	}
	return result, nil
}

func (s *ProfileStore) save(path string) error {
	if path == "" {
		return errors.New("profile path was unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
