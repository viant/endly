package endly

import (
	"fmt"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox"
)

func transformWithUDF(context *Context, udfName, source string, payload interface{}) (interface{}, error) {
	var state = context.state
	var udf, has = UdfRegistry[udfName]
	if !has {
		if candidate, ok := state[udfName]; ok {
			udf, has = candidate.(func(source interface{}, state data.Map) (interface{}, error))
		}
	}
	if !has {
		return nil, fmt.Errorf("failed to lookup udf: %v for: %v", udfName, source)
	}
	transformed, err := udf(payload, state)
	if err != nil {
		return nil, fmt.Errorf("failed to run udf: %v, %v", udfName, err)
	}
	return transformed, nil
}



//DateOfBirth returns formated date of birth
func DateOfBirth(source interface{}, state data.Map) (interface{}, error) {

	aSlice, err := toolbox.JSONToSlice(source)
	fmt.Println("v% aSlice=",aSlice)
	if err != nil {
		return nil, err
	}
	return toolbox.NewDateOfBirthrovider().Get(toolbox.NewContext(), aSlice...)
}