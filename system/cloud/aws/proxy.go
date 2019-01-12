package aws

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"reflect"
	"strings"
)

func BuildRoutes(service interface{}, clientProvider func(context *endly.Context) (interface{}, error)) ([]*endly.Route, error) {
	var result = make([]*endly.Route, 0)
	err := toolbox.ScanStructMethods(service, 1, func(method reflect.Method) error {
		signature := toolbox.GetFuncSignature(method.Func.Interface())
		outNum := method.Type.NumOut()
		if len(signature) != 2 || outNum != 2 {
			return nil
		}
		if method.Type.Out(1).Kind() != reflect.Interface {
			return nil
		}
		action := normalizeAction(method.Name)
		route := &endly.Route{
			Action: action,
			RequestInfo: &endly.ActionInfo{
				Description: fmt.Sprintf("%T.%v(%T)", service, method.Name, reflect.New(signature[1].Elem()).Interface()),
			},
			ResponseInfo: &endly.ActionInfo{
				Description: fmt.Sprintf("%T", reflect.New(method.Type.Out(0).Elem()).Interface()),
			},
			RequestProvider: func() interface{} {
				return reflect.New(signature[1].Elem()).Interface()
			},
			ResponseProvider: func() interface{} {
				return reflect.New(method.Type.Out(0).Elem()).Interface()
			},

			Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
				client, err := clientProvider(context)
				if err != nil {
					return nil, err
				}
				output := method.Func.Call([]reflect.Value{reflect.ValueOf(client), reflect.ValueOf(request)})
				errOutput := output[1].Interface()
				if errOutput != nil {
					return nil,  fmt.Errorf("unable to run %v, %v", action, errOutput)
				}
				return output[0].Interface(), nil
			},
		}
		result = append(result, route)
		return nil
	})
	return result, err
}



func normalizeAction(name string) string {
	if index := strings.LastIndex(name, "."); index != -1 {
		name = string(name[index:])
	}
	if index := strings.LastIndex(name, "Input"); index != -1 {
		name = string(name[0:index])
	}
	name = strings.ToLower(string(name[0:1])) + string(name[1:])
	return name
}
