package policy

import (
	"github.com/viant/endly"
	_ "github.com/viant/endly/system/kubernetes/policy/v1beta1"
)

func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})
}
