package endly

import "fmt"

func transformWithUDF(context *Context, udfName, source string, payload interface{}) (interface{}, error) {
	var state = context.state
	var udf, has = UdfRegistry[udfName]
	if !has {
		return nil, fmt.Errorf("failed to lookup udf: %v for: %v", udfName, source)
	}
	transformed, err := udf(payload, state)
	if err != nil {
		return nil, fmt.Errorf("failed to run udf: %v, %v", udfName, err)
	}
	return transformed, nil
}
