package workflow_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
)

type assertionProofRequest struct{ Invalid bool }
type assertionProofResponse struct{ Fail bool }

func (r *assertionProofResponse) Assertion() []*assertly.Validation {
	if r.Fail {
		return []*assertly.Validation{{FailedCount: 1}}
	}
	return []*assertly.Validation{{PassedCount: 1}}
}
func assertionProofContext(t *testing.T) *endly.Context {
	registry := endly.New()
	service := endly.NewAbstractService("assertion-proof")
	service.Register(&endly.Route{Action: "report", RequestProvider: func() any { return &assertionProofRequest{} }, ResponseProvider: func() any { return &assertionProofResponse{} }, Handler: func(_ *endly.Context, in any) (any, error) {
		return &assertionProofResponse{Fail: in.(*assertionProofRequest).Invalid}, nil
	}})
	registry.Register(service)
	ctx := registry.NewContext(nil)
	t.Cleanup(ctx.Close)
	return ctx
}
func TestWorkflowAssertionScopeIsOptInAndDoesNotLeak(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(fmt.Sprint(strict), func(t *testing.T) {
			ctx := assertionProofContext(t)
			seenPasses, seenFailures := 0, 0
			ctx.Listener = func(event msg.Event) {
				if asserted, ok := event.Value().(interface{ Assertion() []*assertly.Validation }); ok {
					for _, v := range asserted.Assertion() {
						if v != nil {
							seenPasses += v.PassedCount
							seenFailures += v.FailedCount
						}
					}
				}
			}
			request := &workflow.RunRequest{Name: "scoped", AssetURL: filepath.Join(t.TempDir(), "scoped.yaml"), SharedState: true, FailOnAssertion: strict, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{
				{Key: "invalid", Value: map[string]any{"action": "assertion-proof:report", "invalid": true}},
				{Key: "continued", Value: map[string]any{"action": "assertion-proof:report", "post": []string{"continuedAfterFailure = true"}}},
			}}}
			var result workflow.RunResponse
			err := endly.Run(ctx, request, &result)
			if strict {
				require.ErrorContains(t, err, "1 validation assertions failed")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, "true", ctx.State()["continuedAfterFailure"], "strict scopes collect all assertions without skipping later checks")
			next := &workflow.RunRequest{Name: "next", AssetURL: filepath.Join(t.TempDir(), "next.yaml"), SharedState: true, FailOnAssertion: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "valid", Value: map[string]any{"action": "assertion-proof:report"}}}}}
			result = workflow.RunResponse{}
			require.NoError(t, endly.Run(ctx, next, &result))
			require.Zero(t, result.FailedAssertions)
			require.Equal(t, 1, result.PassedAssertions)
			require.Equal(t, 2, seenPasses, "caller still receives every assertion event")
			require.Equal(t, 1, seenFailures)
		})
	}
}
func TestWorkflowAssertionScopeIncludesNestedWorkflow(t *testing.T) {
	ctx := assertionProofContext(t)
	root := t.TempDir()
	child := filepath.Join(root, "child.yaml")
	require.NoError(t, os.WriteFile(child, []byte("pipeline:\n  invalid:\n    action: assertion-proof:report\n    invalid: true\n"), 0600))
	request := &workflow.RunRequest{Name: "parent", AssetURL: filepath.Join(root, "parent.yaml"), SharedState: true, FailOnAssertion: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "nested", Value: map[string]any{"action": "workflow:run", "URL": child, "sharedState": true}}}}}
	var result workflow.RunResponse
	require.ErrorContains(t, endly.Run(ctx, request, &result), "1 validation assertions failed")
}
