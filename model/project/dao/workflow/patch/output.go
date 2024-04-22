package patch

import (
	"github.com/viant/xdatly/handler/response"
	"github.com/viant/xdatly/types/core"
	"github.com/viant/xdatly/types/custom/checksum"
	"reflect"
)

func init() {
	core.RegisterType(PackageName, "Output", reflect.TypeOf(Output{}), checksum.GeneratedTime)

}

type Output struct {
	response.Status `parameter:",kind=output,in=status" anonymous:"true"`
	Data            *Workflow `parameter:",kind=body"`
}
