package eval

import "github.com/viant/toolbox/data"

type Compute func(state data.Map) (interface{}, bool,  error)
