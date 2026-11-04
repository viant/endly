package workflow_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	_ "github.com/viant/endly/service/testing/validator"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
)

type readinessInspectRequest struct{}
type readinessOpenRequest struct{}
type readinessReadRequest struct{}

// These fixtures exercise workflow branching, not an OS app or mailbox adapter.
// A missing process/window observation must never become permission to launch.
func TestWhenUsesPriorOutputPathsForConditionalReadiness(t *testing.T) {
	for _, mode := range []string{"ready", "stopped", "unknown", "unknown-ready", "missing", "empty", "not-ready", "launch-unknown", "launch-failed", "launch-running", "still-stopped"} {
		t.Run(mode, func(t *testing.T) {
			registry := endly.New()
			service := endly.NewAbstractService("readiness-fixture")
			probes, launches, reads := 0, 0, 0
			register := func(name string, request func() any, handle func() map[string]any) {
				service.Register(&endly.Route{Action: name, RequestProvider: request, ResponseProvider: func() any { return &map[string]any{} }, Handler: func(*endly.Context, any) (any, error) { return handle(), nil }})
			}
			register("inspect", func() any { return &readinessInspectRequest{} }, func() map[string]any {
				probes++
				if mode == "missing" {
					return map[string]any{"StructuredContent": map[string]any{}}
				}
				if mode == "empty" {
					return map[string]any{"StructuredContent": map[string]any{"apps": []any{}}}
				}
				state, ready := "running", true
				if mode == "unknown" || mode == "unknown-ready" {
					state, ready = "unknown", mode == "unknown-ready"
				} else if mode == "not-ready" {
					ready = false
				} else if mode != "ready" && (launches == 0 || mode == "still-stopped") {
					state, ready = "stopped", false
				}
				return map[string]any{"StructuredContent": map[string]any{"apps": []any{map[string]any{"state": state, "ready": ready}}}}
			})
			register("open", func() any { return &readinessOpenRequest{} }, func() map[string]any {
				launches++
				status := "succeeded"
				switch mode {
				case "launch-unknown":
					status = "unknown"
				case "launch-failed":
					status = "failed"
				case "launch-running":
					status = "running"
				}
				return map[string]any{"StructuredContent": map[string]any{"executionStatus": status}}
			})
			register("read", func() any { return &readinessReadRequest{} }, func() map[string]any { reads++; return map[string]any{"count": 2} })
			registry.Register(service)
			ctx := registry.NewContext(toolbox.NewContext())
			defer ctx.Close()
			// Reused contexts may retain successful results from a prior invocation.
			ctx.State()["open"] = map[string]any{"StructuredContent": map[string]any{"executionStatus": "succeeded"}}
			ctx.State()["read"] = map[string]any{"count": 2}
			steps := []*model.MapEntry{
				{Key: "resetInvocation", Value: map[string]any{"action": "nop", "init": []string{"currentLaunchStatus = notStarted", "currentReadStatus = notStarted"}}},
				{Key: "probe", Value: map[string]any{"action": "readiness-fixture:inspect", "post": []string{"currentApp = $StructuredContent"}}},
				{Key: "open", Value: map[string]any{"when": "$currentApp.apps[0].state:stopped", "action": "readiness-fixture:open", "post": []string{"currentLaunchStatus = $StructuredContent.executionStatus"}}},
				{Key: "recheck", Value: map[string]any{"when": "$currentLaunchStatus:succeeded", "action": "readiness-fixture:inspect", "post": []string{"currentApp = $StructuredContent"}}},
				{Key: "read", Value: map[string]any{"when": "$currentApp.apps[0].state:running && $currentApp.apps[0].ready:true", "action": "readiness-fixture:read", "post": []string{"currentReadStatus = succeeded"}}},
				{Key: "requireReady", Value: map[string]any{"action": "validator:assert", "actual": map[string]any{"status": "$currentReadStatus", "count": "$read.count"}, "expect": map[string]any{"status": "succeeded", "count": 2}}},
			}
			var response workflow.RunResponse
			err := endly.Run(ctx, &workflow.RunRequest{Name: "conditional-readiness", AssetURL: filepath.Join(t.TempDir(), "workflow.yaml"), SharedState: true, FailOnAssertion: true, Inlined: &model.Inlined{Pipeline: steps}}, &response)
			if mode == "ready" || mode == "stopped" {
				require.NoError(t, err)
				require.Zero(t, response.FailedAssertions)
			} else {
				require.ErrorContains(t, err, "validation assertions failed")
			}
			wantLaunches, wantProbes, wantReads := 0, 1, 0
			switch mode {
			case "ready":
				wantReads = 1
			case "stopped":
				wantLaunches, wantProbes, wantReads = 1, 2, 1
			case "still-stopped":
				wantLaunches, wantProbes = 1, 2
			case "launch-unknown", "launch-failed", "launch-running":
				wantLaunches = 1
			}
			require.Equal(t, wantLaunches, launches, "launch is never replayed or inferred from absent evidence")
			require.Equal(t, wantProbes, probes, "warm readiness does not require repeated discovery")
			require.Equal(t, wantReads, reads, "dependent work requires independently observed readiness")
		})
	}
}
