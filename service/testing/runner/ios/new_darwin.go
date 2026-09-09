//go:build darwin

package ios

import (
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func New() endly.Service {
	return newService(mobile.OSRunner{})
}
