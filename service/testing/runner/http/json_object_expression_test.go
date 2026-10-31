package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONBodyAsMapExpressionPreservesObjectAndArrayTypes(t *testing.T) {
	const source = `{"params":{},"arguments":{"object":{},"array":[],"enabled":false,"zero":0,"nothing":null,"nested":[{"settings":{},"values":[]}]}}`
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	ctx := newContextWithState(map[string]interface{}{"requestJSON": source, "argumentsJSON": `{"object":{},"array":[],"enabled":false,"zero":0,"nothing":null,"nested":[{"settings":{},"values":[]}]}`})
	defer ctx.Close()
	for _, body := range []interface{}{
		"$AsMap($requestJSON)",
		map[string]interface{}{"params": "$AsMap('{}')", "arguments": "$AsMap($argumentsJSON)"},
	} {
		value, err := ctx.AsRequest(ServiceID, "send", map[string]interface{}{
			"requests": []interface{}{map[string]interface{}{
				"URL": server.URL, "Method": "POST", "JSONBody": body,
			}},
		})
		require.NoError(t, err)
		request := value.(*SendRequest)
		require.NoError(t, request.Init())
		_, err = newServiceForTest().send(ctx, request)
		require.NoError(t, err)
		require.JSONEq(t, source, received)
	}
}
