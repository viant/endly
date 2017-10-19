package endly

import "github.com/viant/endly/common"

//UdfRegistry represents a udf registry
var UdfRegistry = make(map[string]func(source interface{}, state common.Map) (interface{}, error))
