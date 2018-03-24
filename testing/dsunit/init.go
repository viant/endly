package dsunit

import (
	"github.com/viant/endly"
	"fmt"
	"github.com/viant/toolbox"
)

func init() {
	endly.Registry.Register(func() endly.Service {
		service := New()

		a, b, c := toolbox.CallerInfo(3)
		fmt.Printf("new dsunit service %v -, %v %v %v\n", &service, a, b, c)

		return service
	})
	endly.UdfRegistry["AsTableRecords"] = AsTableRecords
}
