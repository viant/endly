package docker

import "github.com/viant/endly"

func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})
}
