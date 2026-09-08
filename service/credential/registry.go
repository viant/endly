package credential

import (
	"strings"
	"sync"
)

// Registry maps credential aliases to secret URLs (for example op:// references).
// The map is populated from the root workflow's credentialMap; it is empty until Set is called.
type Registry struct {
	mu          sync.RWMutex
	credentials map[string]string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{credentials: make(map[string]string)}
}

// Set replaces the registry contents with m. Empty aliases or URLs are skipped.
func (r *Registry) Set(m map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.credentials = make(map[string]string, len(m))
	for alias, url := range m {
		if alias == "" || url == "" {
			continue
		}
		r.credentials[alias] = url
	}
}

// HasMap reports whether at least one alias is registered.
func (r *Registry) HasMap() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.credentials) > 0
}

// Resolve returns the secret URL for alias. URLs and unmapped aliases pass through unchanged.
func (r *Registry) Resolve(alias string) (string, error) {
	if alias == "" || strings.Contains(alias, "://") {
		return alias, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if url, ok := r.credentials[alias]; ok {
		return url, nil
	}
	return alias, nil
}
