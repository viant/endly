package buildin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
)

func TestCatResolvesProjectRelativeResourceFromOwner(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, "workflows")
	assetDir := filepath.Join(root, "assets")
	require.NoError(t, os.MkdirAll(workflowDir, 0o700))
	require.NoError(t, os.MkdirAll(assetDir, 0o700))
	asset := filepath.Join(assetDir, "user.txt")
	require.NoError(t, os.WriteFile(asset, []byte("hello"), 0o600))
	state := data.NewMap()
	state.Put(OwnerURL, "file://"+filepath.Join(workflowDir, "test.yaml"))

	value, err := Cat("assets/user.txt", state)
	require.NoError(t, err)
	require.Equal(t, "hello", value)
	found, err := HasResource("assets/user.txt", state)
	require.NoError(t, err)
	require.Equal(t, true, found)
}
