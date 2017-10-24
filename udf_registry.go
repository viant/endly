package endly

import "github.com/viant/toolbox/data"

//UdfRegistry represents a udf registry
var UdfRegistry = make(map[string]func(source interface{}, state data.Map) (interface{}, error))
