package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

type expandedRouteRequest struct {
	Value     string
	Templates map[string]any
}
type rawRouteRequest struct{ Value string }

func TestDirectActionExpansionIsExplicitAndKeepsDeferredFields(t *testing.T) {
	var expanded *expandedRouteRequest
	var raw *rawRouteRequest
	factory := func() endly.Manager {
		registry := endly.New()
		service := endly.NewAbstractService("expansion")
		service.Register(&endly.Route{Action: "expanded", ExpandRequest: true, DeferredRequestFields: []string{"templates"}, RequestProvider: func() any { return &expandedRouteRequest{} }, ResponseProvider: func() any { return &expandedRouteRequest{} }, Handler: func(_ *endly.Context, req any) (any, error) {
			expanded = req.(*expandedRouteRequest)
			return expanded, nil
		}})
		service.Register(&endly.Route{Action: "raw", RequestProvider: func() any { return &rawRouteRequest{} }, ResponseProvider: func() any { return &rawRouteRequest{} }, Handler: func(_ *endly.Context, req any) (any, error) { raw = req.(*rawRouteRequest); return raw, nil }})
		registry.Register(service)
		return registry
	}
	runtime := New(factory)
	defer runtime.Shutdown(context.Background())
	session, err := runtime.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	state := runtime.sessions[session.SessionID].context.State()
	state["value"] = "expanded"
	calls := 0
	state["Counter"] = func(any, data.Map) (any, error) { calls++; return calls, nil }
	for _, action := range []string{"raw", "expanded"} {
		request := map[string]any{"value": "$value"}
		if action == "expanded" {
			request["templates"] = map[string]any{"name": "$Counter()"}
		}
		op, err := runtime.StartAction(&RunActionRequest{SessionID: session.SessionID, Service: "expansion", Action: action, Request: request})
		require.NoError(t, err)
		op, err = runtime.WaitOperation(context.Background(), session.SessionID, op.ID)
		require.NoError(t, err)
		require.Equal(t, OperationSucceeded, op.Status, op.Error)
	}
	require.Equal(t, "$value", raw.Value)
	require.Equal(t, "expanded", expanded.Value)
	require.Equal(t, "$Counter()", expanded.Templates["name"])
	require.Zero(t, calls)
}
