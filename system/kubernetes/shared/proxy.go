package shared

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/kubernetes/registry"
	"github.com/viant/toolbox"
	"reflect"
	"strings"
)

type ServiceHolder struct {
	Name  string
	Impl  interface{}
	IFace reflect.Type
}

func BuildRoutes(service interface{}, clientPrefix string) ([]*endly.Route, error) {
	var result = make([]*endly.Route, 0)

	var services = make(map[string]*ServiceHolder)
	err := toolbox.ScanStructMethods(service, 1, func(method reflect.Method) error {
		if method.Type.NumOut() != 1 {
			return nil
		}
		returnType := method.Type.Out(0)
		candidate := returnType.String()
		if ! strings.HasSuffix(candidate, "Interface") {
			return nil
		}
		holder := &ServiceHolder{
			Name:  method.Name,
			IFace: method.Type.Out(0),
		}
		var result []interface{}
		if method.Type.NumIn() == 2 {
			if method.Type.In(1).Kind() != reflect.String {
				return nil
			}
			result = toolbox.CallFunction(method.Func.Interface(), service, "")
		} else {
			result = toolbox.CallFunction(method.Func.Interface(), service)
		}
		resultValue := reflect.ValueOf(result[0])
		if resultValue.IsNil() {
			return nil
		}
		holder.Impl = result[0]
		services[method.Name] = holder
		return nil
	})

	for i := range services {
		holder := services[i]
		err = toolbox.ScanStructMethods(holder.Impl, 1, func(method reflect.Method) error {
			ifaceTypeName := holder.IFace.String()
			id := ifaceTypeName + "." + method.Name
			adapter, has := registry.Get(id)
			if ! has {
				return nil
			}
			requestType := reflect.ValueOf(adapter).Type().Elem()
			responseType := method.Type.Out(0)
			if responseType.Kind() == reflect.Ptr {
				responseType = responseType.Elem()
			}
			action := actionName(holder, method)
			route := &endly.Route{
				Action:       action,
				OnRawRequest: InitClient,
				RequestInfo: &endly.ActionInfo{
					Description: fmt.Sprintf("%s.%v(%T)", removeNamespace(ifaceTypeName), method.Name, adapter),
				},
				ResponseInfo: &endly.ActionInfo{
					Description: fmt.Sprintf("%s", method.Type.Out(0)),
				},
				RequestProvider: func() interface{} {
					return reflect.New(requestType).Interface()
				},
				ResponseProvider: func() interface{} {
					return reflect.New(responseType).Interface()
				},
				Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
					adapter := request.(registry.ContractAdapter)
					clientCtx, err := GetCtxClient(context)
					if err != nil {
						return nil, err
					}

					service, err := getServce(clientCtx, clientPrefix, holder.Name)
					if err != nil {
						return nil, err
					}
					if err = adapter.SetService(service); err != nil {
						return nil, err
					}
					result, err := adapter.Call()
					if err != nil {
						return nil, err
					}

					resultMap := make(map[string]interface{})
					eventValue := result
					if err = toolbox.DefaultConverter.AssignConverted(&resultMap, result); err == nil {
						resultMap = toolbox.DeleteEmptyKeys(resultMap)
						eventValue = resultMap
					}
					context.Publish(NewOutputEvent(method.Name, "proxy", eventValue))
					return result, nil
				},
			}
			result = append(result, route)
			return nil
		})
	}

	return result, err
}

func getServce(clientCtx *CtxClient, clientPrefix string, kindService string) (interface{}, error) {
	clientID := clientPrefix + clientCtx.ApiVersion
	clientset, err := clientCtx.Clientset()
	if err != nil {
		return nil, err
	}
	clientSetValue := reflect.ValueOf(clientset)
	methodType, ok := clientSetValue.Type().MethodByName(clientID)
	if ! ok {
		return nil, fmt.Errorf("failed to locate %v", clientID)
	}
	getClientMethod := clientSetValue.MethodByName(clientID).Interface()
	results := toolbox.CallFunction(getClientMethod)
	clientValue := reflect.ValueOf(results[0])
	methodType, ok = clientValue.Type().MethodByName(kindService)
	if ! ok {
		return nil, fmt.Errorf("failed to locate %v.%v", clientID, kindService)
	}

	getServiceMethod := clientValue.MethodByName(kindService)
	if methodType.Type.NumIn() == 1 {
		results = toolbox.CallFunction(getServiceMethod.Interface())
	} else {
		results = toolbox.CallFunction(getServiceMethod.Interface(), "")
	}
	return results[0], nil
}

func actionName(holder *ServiceHolder, method reflect.Method) string {
	interfaceType := holder.IFace.String()
	kindName := kindName(interfaceType)
	var action = ""
	if method.Name == "List" {
		action = method.Name + holder.Name
	} else {
		action = method.Name + kindName
	}
	return toolbox.ToCaseFormat(action, toolbox.CaseUpperCamel, toolbox.CaseLowerCamel)
}

func removeNamespace(name string) string {
	isPointer := strings.HasPrefix(name, "*")
	if index := strings.Index(name, "."); index != -1 {
		name = string(name[index+1:])
		if isPointer {
			name = "*" + name
		}
	}
	return name
}

func kindName(name string) string {
	name = strings.Replace(name, "Interface", "", 1)
	return removeNamespace(name)
}
