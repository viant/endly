package exec

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

func TestRunKeepsExistingDottedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	root := t.TempDir()
	directory := filepath.Join(root, "checkout.with.dots")
	require.NoError(t, os.Mkdir(directory, 0755))
	physical, err := filepath.EvalSymlinks(directory)
	require.NoError(t, err)
	for _, mode := range []string{"command", "quoted-command", "options"} {
		t.Run(mode, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			t.Cleanup(ctx.Close)
			request := NewRunRequest(nil, false, "pwd -P")
			switch mode {
			case "command":
				request = NewRunRequest(nil, false, "cd "+directory, "pwd -P")
			case "quoted-command":
				request = NewRunRequest(nil, false, "cd '"+directory+"'", "pwd -P")
			case "options":
				request.Directory = directory
			}
			request.StrictCompletion = true
			request.CheckError = true
			result := New().Run(ctx, request)
			require.NoError(t, result.Err)
			require.Equal(t, physical, strings.TrimSpace(result.Response.(*RunResponse).Output))
		})
	}
}

func TestRunRetainsFileDirectoryFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	root := t.TempDir()
	file := filepath.Join(root, "run.yaml")
	require.NoError(t, os.WriteFile(file, []byte("pipeline: {}\n"), 0600))
	physical, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	request := NewRunRequest(nil, false, "cd "+file, "pwd -P")
	request.StrictCompletion = true
	result := New().Run(ctx, request)
	require.NoError(t, result.Err)
	require.Equal(t, physical, strings.TrimSpace(result.Response.(*RunResponse).Output))
}
