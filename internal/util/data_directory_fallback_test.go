package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/model/location"
)

func TestResourceFallbackSkipsDirectories(t *testing.T) {
	for _, decoded := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "decoded"}[decoded], func(t *testing.T) {
			root := t.TempDir()
			defaults := filepath.Join(root, "default")
			require.NoError(t, os.MkdirAll(filepath.Join(defaults, "rule/platform/planning"), 0755))
			require.NoError(t, os.MkdirAll(filepath.Join(root, "rule/platform"), 0755))
			path := filepath.Join(root, "rule/platform/planning.yaml")
			require.NoError(t, os.WriteFile(path, []byte("name: planning\n"), 0644))
			var text string
			value := map[string]interface{}{}
			var target interface{} = &text
			if decoded {
				target = &value
			}
			resource, err := LoadResourceFromBaseURLs([]string{location.NewResource(defaults).URL, location.NewResource(root).URL}, "@rule/platform/planning", target)
			require.NoError(t, err)
			require.True(t, strings.HasSuffix(resource.URL, "/rule/platform/planning.yaml"))
			if decoded {
				require.Equal(t, "planning", value["name"])
			} else {
				require.Equal(t, "name: planning\n", text)
			}
		})
	}
}

func TestResourceFallbackPreservesInvalidFileError(t *testing.T) {
	root := t.TempDir()
	defaults := filepath.Join(root, "default")
	require.NoError(t, os.MkdirAll(defaults, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(defaults, "planning.json"), []byte("{invalid"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "planning.yaml"), []byte("name: fallback\n"), 0644))
	value := map[string]interface{}{}
	_, err := LoadResourceFromBaseURLs([]string{location.NewResource(defaults).URL, location.NewResource(root).URL}, "@planning", &value)
	require.Error(t, err)
	require.False(t, IsNotSuchResourceError(err), "a malformed file must not silently fall back")
}
