package extensions

import (
	"github.com/viant/endly"
	_ "github.com/viant/endly/system/kubernetes/extensions/v1beta1"
)

func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})
}
