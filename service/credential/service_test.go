package credential

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/scy/cred/secret"
)

func TestService_ResolveAliasToFile(t *testing.T) {
	dir := t.TempDir()
	credFile := filepath.Join(dir, "gcp.json")
	require.NoError(t, os.WriteFile(credFile, []byte(`{
  "type": "service_account",
  "project_id": "viant-e2e",
  "private_key_id": "abc",
  "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
  "client_email": "test@viant-e2e.iam.gserviceaccount.com"
}`), 0o600))

	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  gcp-e2e: `+credFile+`
`), 0o600))

	t.Setenv(credentialsFileEnv, mapping)

	svc := NewService()
	generic, err := svc.GetCredentials(context.Background(), "gcp-e2e")
	require.NoError(t, err)
	require.Equal(t, "viant-e2e", generic.ProjectID)
	require.Equal(t, "test@viant-e2e.iam.gserviceaccount.com", generic.ClientEmail)
}

func TestService_ExpandAlias(t *testing.T) {
	dir := t.TempDir()
	credFile := filepath.Join(dir, "gcp.json")
	payload := `{"type":"service_account","project_id":"viant-e2e","client_email":"test@viant-e2e.iam.gserviceaccount.com"}`
	require.NoError(t, os.WriteFile(credFile, []byte(payload), 0o600))

	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  viant-e2e: `+credFile+`
`), 0o600))

	t.Setenv(credentialsFileEnv, mapping)

	svc := NewService()
	expanded, err := svc.Expand(context.Background(), "echo '${gcp.Data}'", map[secret.Key]secret.Resource{
		"gcp": "viant-e2e",
	})
	require.NoError(t, err)
	require.Contains(t, expanded, `"project_id":"viant-e2e"`)
	require.Contains(t, expanded, `"client_email":"test@viant-e2e.iam.gserviceaccount.com"`)
}

func TestService_MappingPrecedenceOverLegacySecret(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	secretDir := filepath.Join(home, ".secret")
	require.NoError(t, os.MkdirAll(secretDir, 0o700))
	t.Setenv("HOME", home)

	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "viant-e2e.json"), []byte(`{
  "type": "service_account",
  "project_id": "legacy-from-secret-dir",
  "client_email": "legacy@viant-e2e.iam.gserviceaccount.com"
}`), 0o600))

	mappedFile := filepath.Join(dir, "mapped.json")
	require.NoError(t, os.WriteFile(mappedFile, []byte(`{
  "type": "service_account",
  "project_id": "from-mapping-file",
  "client_email": "mapped@viant-e2e.iam.gserviceaccount.com"
}`), 0o600))

	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  viant-e2e: `+mappedFile+`
`), 0o600))
	t.Setenv(credentialsFileEnv, mapping)

	svc := NewService()
	generic, err := svc.GetCredentials(context.Background(), "viant-e2e")
	require.NoError(t, err)
	require.Equal(t, "from-mapping-file", generic.ProjectID)
	require.Equal(t, "mapped@viant-e2e.iam.gserviceaccount.com", generic.ClientEmail)
}
