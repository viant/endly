//go:build !darwin

package ios

import (
	"strings"
	"testing"

	"github.com/viant/endly"
)

func TestUnsupportedServicePreservesActionSurface(t *testing.T) {
	service := New()
	ctx := endly.New().NewContext(nil)
	for _, action := range []string{"doctor", "device-list", "device-lease", "device-release", "build", "open", "attach", "run", "repl", "cleanup"} {
		route, err := service.Route(action)
		if err != nil {
			t.Fatalf("missing stub route %q: %v", action, err)
		}
		response := service.Run(ctx, route.RequestProvider())
		if response.Err == nil || !strings.Contains(response.Err.Error(), "ios:"+action+" is unsupported") {
			t.Fatalf("route %q did not return its unsupported error: %+v", action, response)
		}
	}
}
