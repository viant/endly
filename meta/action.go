package meta

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

//Action represents service action meta
type Action struct {
	*endly.Route
	Request      interface{}
	RequestMeta  *toolbox.StructMeta
	Response     interface{}
	ResponseMeta *toolbox.StructMeta
}
