package endly_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/service/testing/runner/webdriver"
	"github.com/viant/toolbox"
	"testing"
)

func TestRequestConversionDiagnosticRedactsExpandedValues(t *testing.T) {
	for _, sentinel := range []string{"SYNTHETIC_PASSWORD_b4beed_7312", "a"} {
		t.Run(sentinel, func(t *testing.T) {
			ctx := endly.New().NewContext(toolbox.NewContext())
			defer ctx.Close()
			state := ctx.State()
			state.Put("syntheticPassword", sentinel)
			_, err := ctx.AsRequest("webdriver", "run", map[string]interface{}{
				"actions": []interface{}{map[string]interface{}{"calls": []interface{}{
					map[string]interface{}{"method": "SendKeys", "parameters": []interface{}{"$syntheticPassword"}},
					map[string]interface{}{"method": "Click", "parameters": map[string]interface{}{"invalid": []interface{}{}}},
				}}},
			})
			require.Error(t, err)
			if len(sentinel) > 1 {
				require.NotContains(t, err.Error(), sentinel)
			} else {
				require.NotContains(t, err.Error(), "[a]")
			}
			require.Contains(t, err.Error(), "**webdriver.MethodCall")
			require.Contains(t, err.Error(), "*webdriver.RunRequest")
			require.Contains(t, err.Error(), "map[string]interface {} into type *[]interface {}")
		})
	}
}

type panicRequest struct{}

func TestRequestConversionPanicDiagnosticRedactsRequest(t *testing.T) {
	const sentinel = "SYNTHETIC_PANIC_PASSWORD_a9283"
	manager := endly.New()
	service := endly.NewAbstractService("panicConversion")
	service.Register(&endly.Route{
		Action:           "request",
		RequestProvider:  func() interface{} { return &panicRequest{} },
		OnRawRequest:     func(*endly.Context, map[string]interface{}) error { panic("bad request " + sentinel) },
		ResponseProvider: func() interface{} { return &struct{}{} },
		Handler:          func(*endly.Context, interface{}) (interface{}, error) { return nil, nil },
	})
	manager.Register(service)
	ctx := manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	_, err := ctx.NewRequest("panicConversion", "request", map[string]interface{}{"parameters": []interface{}{sentinel}})
	require.Error(t, err)
	require.NotContains(t, err.Error(), sentinel)
	require.Contains(t, err.Error(), "bad request [REDACTED]")
	require.NotContains(t, err.Error(), "parameters")
}
