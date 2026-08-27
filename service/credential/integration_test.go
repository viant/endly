//go:build integration

package credential

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/viant/afsc/op"
)

// TestIntegrationResolveOpURL exercises alias resolution through a real 1Password secret.
//
// Run manually:
//
//	op signin
//	export OP_INTEGRATION_REF='op://Private/gcp-e2e.json/notesPlain'
//	go test ./service/credential/... -tags=integration -run TestIntegrationResolveOpURL -count=1 -v
func TestIntegrationResolveOpURL(t *testing.T) {
	ref := os.Getenv("OP_INTEGRATION_REF")
	if ref == "" {
		t.Skip("set OP_INTEGRATION_REF to an op:// secret reference")
	}

	dir := t.TempDir()
	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  gcp-e2e: `+ref+`
`), 0o600))
	t.Setenv(credentialsFileEnv, mapping)

	svc := NewService()
	generic, err := svc.GetCredentials(context.Background(), "gcp-e2e")
	require.NoError(t, err)
	require.NotEmpty(t, generic.ProjectID)
	require.NotEmpty(t, generic.ClientEmail)
}
