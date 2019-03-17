package docker

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"reflect"
)

//BuildRoutes build proxy routes
func BuildRoutes(service interface{}, clientProvider func(context *endly.Context) (*CtxClient, error)) ([]*endly.Route, error) {
	var result = make([]*endly.Route, 0)
	err := toolbox.ScanStructMethods(service, 1, func(method reflect.Method) error {
		if method.PkgPath != "" { //ignore private
			return nil
		}
		action := toolbox.ToCaseFormat(method.Name, toolbox.CaseUpperCamel, toolbox.CaseLowerCamel)
		adaper, ok := registry[action+"Request"]
		if !ok {
			return nil
		}
		if method.Type.NumOut() == 0 {
			return nil
		}

		var resultType interface{}
		if method.Type.NumOut() > 0 {
			resultType = reflect.New(method.Type.Out(0)).Interface()
		}

		route := &endly.Route{
			OnRawRequest: initClient,
			Action:       action,
			RequestInfo: &endly.ActionInfo{
				Description: fmt.Sprintf("%T.%v(%T)", service, method.Name, reflect.New(reflect.TypeOf(adaper).Elem()).Interface()),
			},
			ResponseInfo: &endly.ActionInfo{
				Description: fmt.Sprintf("%T", resultType),
			},
			RequestProvider: func() interface{} {
				return reflect.New(reflect.TypeOf(adaper).Elem()).Interface()
			},
			ResponseProvider: func() interface{} {
				if resultType == nil {
					return nil
				}
				return reflect.New(method.Type.Out(0)).Interface()
			},
			Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
				var result interface{}
				err := runAdapter(context, adaper, &result)
				if err == nil {
					publishEvent(context, method.Name, result)
				}
				return result, err
			},
		}
		result = append(result, route)
		return nil
	})
	return result, err
}
