package model

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/internal/util"
)

func TestResourceFallbackDoesNotUseProcessDirectoryAsInstance(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	foreign := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(foreign, "app.yaml"), []byte("pipeline:\n  wrong:\n    action: nop\n"), 0600))
	require.NoError(t, os.Chdir(foreign))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "default"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "default", "app.yaml"), []byte("pipeline:\n  selected:\n    action: nop\n"), 0600))
	inline := Inlined{baseURL: root}
	attrs := map[string]interface{}{"request": "@app", "action": "run"}
	request := map[string]interface{}{}
	require.NoError(t, inline.loadRequest(attrs, request, nil))
	require.Contains(t, request, "pipeline")
	require.Contains(t, fmt.Sprint(request["pipeline"]), "selected")
	require.NotContains(t, fmt.Sprint(request["pipeline"]), "wrong")
	// A nonexistent instance falls through to default; it never resolves as an
	// empty successful resource because a download error was shadowed.
	var source string
	_, err = util.LoadResourceFromBaseURLs([]string{filepath.Join(root, "absent"), filepath.Join(root, "default")}, "@app", &source)
	require.NoError(t, err)
	require.Contains(t, source, "selected")
	_, err = util.LoadResourceFromBaseURLs([]string{root}, "@missing", &source)
	require.Error(t, err)
}
