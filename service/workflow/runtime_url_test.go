package workflow

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model"
)

func runtimeURLFixture(t *testing.T) (*Service, *endly.Context) {
	t.Helper()
	manager := endly.New()
	service := New().(*Service)
	manager.Register(service)
	ctx := manager.NewContext(nil)
	t.Cleanup(ctx.Close)
	return service, ctx
}

func writeRuntimeURLFixture(t *testing.T, filename, marker string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0700))
	require.NoError(t, os.WriteFile(filename, []byte("pipeline:\n  check:\n    action: nop\n    init: {selected: '"+marker+"'}\n"), 0600))
}

func runRuntimeURLFixture(ctx *endly.Context, source string) error {
	var result RunResponse
	return endly.Run(ctx, &RunRequest{URL: source, SharedState: true}, &result)
}

func TestRuntimeWorkflowURLExecutesWithoutManagementPreload(t *testing.T) {
	for _, mode := range []string{"absolute", "fileURI", "extensionless", "json", "http"} {
		t.Run(mode, func(t *testing.T) {
			service, ctx := runtimeURLFixture(t)
			filename := filepath.Join(t.TempDir(), "generated.yaml")
			writeRuntimeURLFixture(t, filename, mode)
			source := filename
			switch mode {
			case "fileURI":
				source = "file://" + filename
			case "extensionless":
				source = strings.TrimSuffix(filename, ".yaml")
			case "json":
				filename = filepath.Join(t.TempDir(), "generated.json")
				data, err := json.Marshal(&RunRequest{Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "check", Value: map[string]any{"action": "nop", "init": []string{"selected = " + mode}}}}}})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filename, data, 0600))
				source = filename
			case "http":
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/generated.yaml" {
						http.NotFound(w, r)
						return
					}
					_, _ = w.Write([]byte("pipeline:\n  check:\n    action: nop\n    init: {selected: 'http'}\n"))
				}))
				t.Cleanup(peer.Close)
				source = peer.URL + "/generated.yaml"
			}
			request := &RunRequest{URL: source, SharedState: true}
			var result RunResponse
			require.NoError(t, endly.Run(ctx, request, &result))
			require.Equal(t, "generated", request.Name)
			require.Equal(t, mode, ctx.State()["selected"])
			require.False(t, service.HasWorkflow("generated"), "runtime resources stay invocation-local")
		})
	}
}

func TestRuntimeWorkflowURLNestedRelativeKeepsParentSource(t *testing.T) {
	service, ctx := runtimeURLFixture(t)
	root := t.TempDir()
	writeRuntimeURLFixture(t, filepath.Join(root, "child", "run.yaml"), "child")
	parent := filepath.Join(root, "run.yaml")
	require.NoError(t, os.WriteFile(parent, []byte("pipeline:\n  child:\n    action: workflow:run\n    URL: child/run.yaml\n    sharedState: true\n  check:\n    action: nop\n    init: {parentReached: true}\n"), 0600))
	require.NoError(t, runRuntimeURLFixture(ctx, parent))
	require.Equal(t, "child", ctx.State()["selected"])
	require.Equal(t, true, ctx.State()["parentReached"])
	require.False(t, service.HasWorkflow("run"), "neither same-basename graph replaces a registry entry")
}

func TestRuntimeWorkflowURLReloadsAndDoesNotShadowRegisteredName(t *testing.T) {
	service, ctx := runtimeURLFixture(t)
	registered := &RunRequest{Name: "same", AssetURL: filepath.Join(t.TempDir(), "registered.yaml"), Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "check", Value: map[string]any{"action": "nop", "init": []string{"selected = registered"}}}}}}
	require.NoError(t, registered.Init())
	require.NoError(t, service.Register(registered.workflow))
	original, err := service.Workflow("same")
	require.NoError(t, err)
	a, b := filepath.Join(t.TempDir(), "same.yaml"), filepath.Join(t.TempDir(), "same.yaml")
	reused := &RunRequest{SharedState: true}
	for _, item := range []struct{ filename, marker string }{{a, "first"}, {b, "second"}, {a, "rewritten"}} {
		writeRuntimeURLFixture(t, item.filename, item.marker)
		reused.URL = item.filename
		var result RunResponse
		require.NoError(t, endly.Run(ctx, reused, &result))
		require.Equal(t, item.marker, ctx.State()["selected"])
		current, err := service.Workflow("same")
		require.NoError(t, err)
		require.Same(t, original, current)
	}
	var result RunResponse
	require.NoError(t, endly.Run(ctx, &RunRequest{Name: "same", SharedState: true}, &result))
	require.Equal(t, "registered", ctx.State()["selected"])
}

func TestRuntimeWorkflowURLMissingRetainsTypedResourceCause(t *testing.T) {
	service, ctx := runtimeURLFixture(t)
	source := filepath.Join(t.TempDir(), "missing.yaml")
	request := &RunRequest{URL: source}
	require.NoError(t, request.Init())
	_, err := service.getWorkflow(ctx, request)
	require.Error(t, err)
	var missing *util.NotSuchResourceError
	require.True(t, errors.As(err, &missing), "preserve the canonical loader's typed missing-resource cause")
	require.Contains(t, err.Error(), "missing.yaml")
}

func TestRuntimeWorkflowURLPreservesInstanceDefaultResourceFallback(t *testing.T) {
	_, ctx := runtimeURLFixture(t)
	root := t.TempDir()
	instance := filepath.Join(root, "instances", "001_case")
	require.NoError(t, os.MkdirAll(instance, 0700))
	parent := filepath.Join(root, "run.yaml")
	require.NoError(t, os.WriteFile(parent, []byte("pipeline:\n  cases:\n    subPath: instances/${index}_*\n    template:\n      run:\n        action: run\n        sharedState: true\n        request: '@test'\n"), 0600))
	paths := []string{filepath.Join(instance, "test.yaml"), filepath.Join(root, "default", "test.yaml"), filepath.Join(root, "test.yaml")}
	for i, marker := range []string{"instance", "default", "resource"} {
		writeRuntimeURLFixture(t, paths[i], marker)
	}
	for i, marker := range []string{"instance", "default", "resource"} {
		require.NoError(t, runRuntimeURLFixture(ctx, parent))
		require.Equal(t, marker, ctx.State()["selected"])
		require.NoError(t, os.Remove(paths[i]))
	}
}
