package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNestedRequestKeepsRootFallbackAndSpecificPrecedence(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() { require.NoError(t, os.Chdir(previous)) })
	request := "regression/database/ci_ads/setup.json"
	specific := filepath.Join(root, "regression/database")
	write := func(base, value string) {
		path := filepath.Join(base, request)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(`{"selected":"`+value+`"}`), 0600))
	}
	write(root, "root")
	load := func(instance string) map[string]interface{} {
		p := Inlined{baseURL: specific, tagPathURL: instance}
		attributes := map[string]interface{}{"action": "run", "request": "@regression/database/ci_ads/setup"}
		actual := map[string]interface{}{}
		err := p.loadRequest(attributes, actual, nil)
		require.NoError(t, err)
		return actual
	}
	require.Equal(t, "root", load("")["selected"])
	write(specific, "workflow")
	require.Equal(t, "workflow", load("")["selected"])
	write(filepath.Join(specific, "default"), "default")
	require.Equal(t, "default", load("")["selected"])
	instance := filepath.Join(root, "instance")
	write(instance, "instance")
	require.Equal(t, "instance", load(instance)["selected"])
}
