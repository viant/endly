package git

import "github.com/viant/endly"

func init() {
	endly.Registry.Register(func() endly.Service {
		return New()
	})
}
