package endly


//UdfRegistry represents a udf registry
var UdfRegistry =  make(map[string]func(source interface{}) (interface{}, error))
