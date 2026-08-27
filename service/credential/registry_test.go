package credential

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_Resolve(t *testing.T) {
	dir := t.TempDir()
	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  gcp-e2e: op://Private/gcp-e2e.json/notesPlain
`), 0o600))

	t.Setenv(credentialsFileEnv, mapping)

	registry := NewRegistry()

	url, err := registry.Resolve("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "op://Private/gcp-e2e.json/notesPlain", url)

	url, err = registry.Resolve("mysql")
	require.NoError(t, err)
	require.Equal(t, "mysql", url)

	url, err = registry.Resolve("op://already/a/url")
	require.NoError(t, err)
	require.Equal(t, "op://already/a/url", url)
}

func TestRegistry_ResolvePassThroughWhenAliasNotInMapping(t *testing.T) {
	dir := t.TempDir()
	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  other: file:///tmp/x.json
`), 0o600))

	t.Setenv(credentialsFileEnv, mapping)

	registry := NewRegistry()
	url, err := registry.Resolve("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "gcp-e2e", url)
}

func TestRegistry_ResolvePassThroughWithoutMapping(t *testing.T) {
	t.Setenv(credentialsFileEnv, "")

	registry := NewRegistry()
	url, err := registry.Resolve("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "gcp-e2e", url)
}

func TestRegistry_ResolveErrorWhenMappingFileInvalid(t *testing.T) {
	t.Setenv(credentialsFileEnv, filepath.Join(t.TempDir(), "missing.yaml"))

	registry := NewRegistry()
	_, err := registry.Resolve("gcp-e2e")
	require.Error(t, err)
	require.Contains(t, err.Error(), credentialsFileEnv)
}

func TestRegistry_MappingPrecedenceOverLegacySecret(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	secretDir := filepath.Join(home, ".secret")
	require.NoError(t, os.MkdirAll(secretDir, 0o700))
	t.Setenv("HOME", home)
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "gcp-e2e.json"), []byte(`{}`), 0o600))

	mappedURL := filepath.Join(dir, "mapped.json")
	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  gcp-e2e: `+mappedURL+`
`), 0o600))
	t.Setenv(credentialsFileEnv, mapping)

	registry := NewRegistry()
	url, err := registry.Resolve("gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, mappedURL, url)
}
