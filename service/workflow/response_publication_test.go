package workflow_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

type literalResponseRequest struct{}

func TestCompletedActionResponsesRemainConcreteWhileVariablesExpand(t *testing.T) {
	registry := endly.New()
	service := endly.NewAbstractService("literal-response")
	service.Register(&endly.Route{Action: "produce", RequestProvider: func() any { return &literalResponseRequest{} }, ResponseProvider: func() any { return &map[string]any{} }, Handler: func(*endly.Context, any) (any, error) {
		return map[string]any{"number": 7, "value": "$Counter()", "nested": []any{map[string]any{"value": "$Counter()"}}}, nil
	}})
	registry.Register(service)
	ctx := registry.NewContext(toolbox.NewContext())
	defer ctx.Close()
	calls := 0
	state := ctx.State()
	state["Counter"] = func(any, data.Map) (any, error) { calls++; return calls, nil }
	var response any
	request := &workflow.RunRequest{Name: "response", AssetURL: filepath.Join(t.TempDir(), "response.yaml"), SharedState: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "produce", Value: map[string]any{"action": "literal-response:produce", "init": []string{"explicitInit = $Counter()"}, "post": []string{"explicitPost = $number"}}}}}}
	require.NoError(t, endly.Run(ctx, request, &response))
	require.Equal(t, 1, calls, "only the authored init UDF is evaluated")
	require.EqualValues(t, 1, state["explicitInit"])
	require.EqualValues(t, 7, state["explicitPost"], "authored post resolves the response field")
	produced := state["produce"].(map[string]any)
	require.Equal(t, "$Counter()", produced["value"])
	require.Equal(t, "$Counter()", produced["nested"].([]any)[0].(map[string]any)["value"])
}
