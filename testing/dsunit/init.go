package dsunit

import "github.com/viant/endly"

func init() {
	endly.Registry.Register(func () endly.Service{
		return New()
	})
	endly.UdfRegistry["AsTableRecords"] = AsTableRecords
}

