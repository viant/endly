package dbunit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	core "github.com/viant/dbunit"
	"github.com/viant/dbunit/fixture"
	"github.com/viant/toolbox/data"
)

func TestHydrationCapturePublicationIsAtomicAndExact(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	state := ctx.State()
	state["blocked"] = "scalar"
	result := s.Run(ctx, &HydrateRequest{Database: "app", Assets: Assets{Capture: "result"}, HydrateRequest: core.HydrateRequest{Tables: []core.HydrationTable{{Name: "records", Source: rowsSource(map[string]any{"id": 1}), Captures: []core.HydrationCapture{{Field: []string{"id"}, Path: "valid.id"}, {Field: []string{"id"}, Path: "blocked.id"}}}}}})
	require.ErrorContains(t, result.Err, "non-object parent")
	require.NotContains(t, state, "valid")
	require.NotContains(t, state, "result")
	require.Equal(t, "scalar", state["blocked"])
	calls := 0
	state["Counter"] = func(any, data.Map) (any, error) { calls++; return calls, nil }
	invalid := s.Run(ctx, &HydrateRequest{Database: "app", Assets: Assets{Capture: "result"}, HydrateRequest: core.HydrateRequest{Tables: []core.HydrationTable{{Name: "records", Source: rowsSource(map[string]any{"id": "$Counter()"}), Captures: []core.HydrationCapture{{Field: []string{"id"}, Path: "side++"}}}}}})
	require.ErrorContains(t, invalid.Err, "literal dotted path")
	require.Zero(t, calls)
	missing := s.Run(ctx, &HydrateRequest{Database: "app", Assets: Assets{Capture: "result"}, HydrateRequest: core.HydrateRequest{Tables: []core.HydrationTable{{Name: "records", Source: rowsSource(map[string]any{"id": 1}), Captures: []core.HydrationCapture{{Field: []string{"id"}, Path: "valid.id"}, {Field: []string{"absent"}, Path: "missing.id"}}}}}})
	require.Error(t, missing.Err)
	require.NotContains(t, state, "valid")
	require.NotContains(t, state, "result")
	state["created"] = map[string]any{"keep": "original", "record": map[string]any{"stale": true}}
	action(t, s, ctx, &HydrateRequest{Database: "app", Assets: Assets{Capture: "result"}, HydrateRequest: core.HydrateRequest{State: data.Map{"counter": 5}, StateCaptures: []core.HydrationStateCapture{{From: "counter", Path: "created.counter"}}, Tables: []core.HydrationTable{{Name: "records", Source: rowsSource(map[string]any{"id": 1, "label": nil}), PostIncrement: []string{"counter"}, Captures: []core.HydrationCapture{{Path: "created.record"}, {Field: []string{"label"}, Path: "created.nullLabel"}}}}}})
	created := state["created"].(map[string]any)
	require.Equal(t, "original", created["keep"])
	require.NotContains(t, created["record"], "stale")
	require.EqualValues(t, 6, created["counter"])
	require.Contains(t, created, "nullLabel")
	require.Nil(t, created["nullLabel"])
	require.NotContains(t, state, "counter")
	selected, err := stateContainer(state, "result.tables")
	require.NoError(t, err)
	require.Len(t, selected, 1)
}

func TestMapActionRetainsExplicitNullDefaultsAndResolvedSources(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE defaults(id INTEGER, nullable TEXT DEFAULT 'database-default')"}}})
	request, err := ctx.AsRequest(ServiceID, "map", map[string]any{"database": "app", "capture": "mapped", "source": map[string]any{"kind": "maps", "data": []any{map[string]any{"id": 1}}}, "mapping": map[string]any{"name": "defaults", "tables": []any{map[string]any{"name": "defaults", "table": "defaults", "columns": []any{map[string]any{"name": "id"}, map[string]any{"name": "nullable", "default": nil}}}}}})
	require.NoError(t, err)
	result := action(t, s, ctx, request).(*core.MapResult)
	require.True(t, result.Datasets[0].Source.Expanded)
	require.Contains(t, result.Datasets[0].Source.Data[0], "nullable")
	require.Nil(t, result.Datasets[0].Source.Data[0]["nullable"])
	action(t, s, ctx, &PrepareRequest{Database: "app", DatasetsFrom: "mapped.datasets"})
	actual := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT nullable FROM defaults"}}).(*core.QueryResult)
	require.Nil(t, actual.Rows[0]["nullable"])
	conflict := s.Run(ctx, &MapRequest{Database: "app", SourceFrom: "mapped.datasets[0].source", MapRequest: core.MapRequest{Source: fixture.Maps([]map[string]any{})}})
	require.ErrorContains(t, conflict.Err, "mutually exclusive")
	for _, name := range []string{"hydrate", "sequence", "map"} {
		route, err := s.Route(name)
		require.NoError(t, err)
		encoded, err := json.Marshal(route.RequestProvider())
		require.NoError(t, err)
		require.Contains(t, string(encoded), "database")
	}
}
