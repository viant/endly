package credential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
)

const (
	credentialsFileEnv  = "E2E_CREDENTIALS_FILE"
	defaultRelativePath = "resource/e2e-credentials.yaml"
)

var e2eAliases = map[string]struct{}{
	"viant-e2e": {},
	"gcp-e2e":   {},
}

type fileConfig struct {
	Credentials map[string]string `yaml:"credentials"`
}

// Registry maps short e2e credential aliases to secret URLs (for example op:// references).
type Registry struct {
	mu          sync.Mutex
	credentials map[string]string
	path        string
	loaded      bool
	loadErr     error
}

// NewRegistry creates an empty registry. Aliases are resolved on first use.
func NewRegistry() *Registry {
	return &Registry{}
}

// MappingConfigured reports whether an e2e credentials mapping file is available.
func MappingConfigured() bool {
	path, err := locateCredentialsFile()
	return err == nil && path != ""
}

// LegacyE2ECredentialsConfigured reports whether ~/.secret has a viant-e2e or gcp-e2e credential file.
func LegacyE2ECredentialsConfigured() bool {
	_, ok := legacyE2ESecretPath("gcp-e2e")
	return ok
}

// Resolve returns the secret URL for alias. Bare aliases such as mysql pass through unchanged.
// When resource/e2e-credentials.yaml (or E2E_CREDENTIALS_FILE) is present, viant-e2e and gcp-e2e
// resolve through that mapping. Otherwise those aliases pass through to scy ~/.secret lookup.
func (r *Registry) Resolve(alias string) (string, error) {
	if alias == "" || strings.Contains(alias, "://") {
		return alias, nil
	}
	r.ensureLoaded()
	if r.loadErr != nil {
		if _, required := e2eAliases[alias]; required {
			return "", fmt.Errorf("e2e credential %q: %w", alias, r.loadErr)
		}
		return alias, nil
	}
	if r.path != "" {
		if url, ok := r.credentials[alias]; ok {
			return url, nil
		}
		if _, required := e2eAliases[alias]; required {
			return "", fmt.Errorf("e2e credential alias %q not defined in %s", alias, r.path)
		}
	}
	return alias, nil
}

func legacyE2ESecretPath(alias string) (string, bool) {
	if _, ok := e2eAliases[alias]; !ok {
		return "", false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	candidates := []string{alias}
	if alias == "gcp-e2e" {
		candidates = append(candidates, "viant-e2e")
	}
	for _, name := range candidates {
		path := filepath.Join(home, ".secret", name+".json")
		if toolbox.FileExists(path) {
			return path, true
		}
	}
	return "", false
}

func (r *Registry) ensureLoaded() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded {
		return
	}
	r.loaded = true
	r.credentials = make(map[string]string)

	path, err := locateCredentialsFile()
	if err != nil {
		r.loadErr = err
		return
	}
	if path == "" {
		return
	}
	r.path = path
	data, err := os.ReadFile(path)
	if err != nil {
		r.loadErr = fmt.Errorf("read credentials mapping %s: %w", path, err)
		return
	}
	var config fileConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		r.loadErr = fmt.Errorf("parse credentials mapping %s: %w", path, err)
		return
	}
	for alias, url := range config.Credentials {
		if alias == "" || url == "" {
			continue
		}
		r.credentials[alias] = url
	}
}

func locateCredentialsFile() (string, error) {
	if path := os.Getenv(credentialsFileEnv); path != "" {
		if !toolbox.FileExists(path) {
			return "", fmt.Errorf("%s=%q does not exist", credentialsFileEnv, path)
		}
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, defaultRelativePath)
		if toolbox.FileExists(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil
}
