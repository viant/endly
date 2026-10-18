package endly_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"testing"
)

type directContainerRequest struct {
	Nested  interface{}
	List    []interface{}
	Literal string
	Mixed   string
	Scalar  string
}

func TestAsRequestWholeUDFReturnedContainers(t *testing.T) {
	manager := endly.New()
	service := endly.NewAbstractService("capture")
	service.Register(&endly.Route{Action: "request", RequestProvider: func() interface{} { return &directContainerRequest{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(*endly.Context, interface{}) (interface{}, error) { return nil, nil }})
	manager.Register(service)
	ctx := manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	state := ctx.State()
	calls := 0
	producedCalls := 0
	canonical := map[string]interface{}{"value": "$OnceCounter()", "empty": []interface{}{}, "literal": "ordinary text"}
	state["Produced"] = func(interface{}, data.Map) (interface{}, error) { producedCalls++; return canonical, nil }
	state["OnceCounter"] = func(_ interface{}, current data.Map) (interface{}, error) { calls++; return current["invocation"], nil }
	state["ScalarProduced"] = func(interface{}, data.Map) (interface{}, error) { return "$OnceCounter()", nil }
	source := map[string]interface{}{"nested": map[string]interface{}{"payload": "$Produced()"}, "list": []interface{}{map[string]interface{}{"payload": "$Produced()"}}, "literal": "ordinary text", "mixed": "prefix $Produced() suffix", "scalar": "$ScalarProduced()"}
	for _, id := range []int{11, 22} {
		state["invocation"] = id
		before := calls
		producedBefore := producedCalls
		request, err := ctx.AsRequest("capture", "request", source)
		require.NoError(t, err)
		actual := request.(*directContainerRequest)
		object := actual.Nested.(map[string]interface{})["payload"].(map[string]interface{})
		require.Equal(t, id, object["value"])
		require.Equal(t, before+2, calls, "exactly one leaf evaluation per exact UDF-produced container")
		require.IsType(t, map[string]interface{}{}, object["empty"])
		require.Equal(t, "ordinary text", actual.Literal)
		require.Contains(t, actual.Mixed, "prefix ")
		require.Contains(t, actual.Mixed, "$OnceCounter()", "mixed string result is not reevaluated")
		require.Equal(t, "$OnceCounter()", actual.Scalar, "scalar UDF result is not normalized or reevaluated")
		require.Equal(t, producedBefore+3, producedCalls, "each original outer UDF occurrence executes once")
		require.Equal(t, "$OnceCounter()", canonical["value"])
		require.IsType(t, []interface{}{}, canonical["empty"])
		require.Equal(t, "$Produced()", source["nested"].(map[string]interface{})["payload"])
		require.Equal(t, "$Produced()", source["list"].([]interface{})[0].(map[string]interface{})["payload"])
	}
}
