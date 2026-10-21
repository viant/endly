package endly_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"testing"
)

type deferredFixtureRequest struct {
	Database string
	Data     map[string]any
}

func TestRouteDeferredFieldsPreserveRawTemplates(t *testing.T) {
	manager := endly.New()
	service := endly.NewAbstractService("deferred")
	service.Register(&endly.Route{Action: "prepare", DeferredRequestFields: []string{"data"}, RequestProvider: func() any { return &deferredFixtureRequest{} }, ResponseProvider: func() any { return &struct{}{} }, Handler: func(*endly.Context, any) (any, error) { return nil, nil }})
	manager.Register(service)
	ctx := manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	state := ctx.State()
	state["db"] = "app"
	calls := 0
	state["Next"] = func(any, data.Map) (any, error) { calls++; return "generated", nil }
	source := map[string]any{"database": "$db", "DaTa": map[string]any{"name": "$Next()"}}
	request, err := ctx.AsRequest("deferred", "prepare", source)
	require.NoError(t, err)
	actual := request.(*deferredFixtureRequest)
	require.Equal(t, "app", actual.Database)
	require.Equal(t, "$Next()", actual.Data["name"])
	require.Zero(t, calls)
	require.Equal(t, "$db", source["database"])
}
