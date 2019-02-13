package shared

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"reflect"
	"strings"
)

//BuildRoutes build routes
func BuildRoutes(service interface{}, apiPrefix string) ([]*endly.Route, error) {
	return BuildRoutesWithPrefix(service, apiPrefix, "")
}

func BuildRoutesWithPrefix(service interface{}, apiPrefix, actionPrefix string) ([]*endly.Route, error) {
	var result = make([]*endly.Route, 0)
	apis, err := buildAPIHolders(service, apiPrefix)
	if err != nil {
		return nil, err
	}
	for i := range apis {
		holder := apis[i]
		err = toolbox.ScanStructMethods(holder.impl, 1, func(method reflect.Method) error {
			ifaceTypeName := holder.iFace.String()
			id := holder.id + "." + method.Name
			adapter, has := Get(id)
			if !has {
				return nil
			}
			requestType := reflect.ValueOf(adapter).Type().Elem()
			responseType := method.Type.Out(0)
			if responseType.Kind() == reflect.Ptr {
				responseType = responseType.Elem()
			}
			action := actionPrefix + actionName(holder, method)
			route := &endly.Route{
				Action:       action,
				OnRawRequest: Init,
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
					adapter := request.(ContractAdapter)
					clientCtx, err := GetCtxClient(context)
					if err != nil {
						return nil, err
					}
					service, err := getService(clientCtx, holder)
					if err != nil {
						return nil, err
					}
					if err = adapter.SetService(service); err != nil {
						return nil, err
					}
					result, err := adapter.Call()
					if err != nil {
						if IsNotFound(err) {
							return &NotFound{Message: err.Error()}, nil
						}
						return nil, err
					}

					resultMap := make(map[string]interface{})
					eventValue := result
					if err = toolbox.DefaultConverter.AssignConverted(&resultMap, result); err == nil {
						resultMap = toolbox.DeleteEmptyKeys(resultMap)
						eventValue = resultMap
					}
					if context.IsLoggingEnabled() {
						context.Publish(NewOutputEvent(method.Name, "proxy", eventValue))
					}
					return result, nil
				},
			}
			result = append(result, route)
			return nil
		})
	}

	return result, err
}

func buildAPIHolders(service interface{}, apiPrefix string) (map[string]*apiHolder, error) {
	var services = make(map[string]*apiHolder)
	err := toolbox.ScanStructMethods(service, 1, func(method reflect.Method) error {
		if method.Type.NumOut() != 1 {
			return nil
		}
		returnType := method.Type.Out(0)
		//candidate := returnType.String()
		if returnType.Kind() != reflect.Interface {
			return nil
		}
		holder := newServiceHolder(strings.ToLower(apiPrefix), method)
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
		holder.impl = result[0]
		services[method.Name] = holder
		return nil
	})
	return services, err
}

func getService(clientCtx *CtxClient, holder *apiHolder) (interface{}, error) {
	apiVersion := strings.Title(holder.apiVersion)
	if strings.Contains(apiVersion, "/") {
		apiVersion = strings.Replace(apiVersion, "/", "", 1)
	} else {
		apiVersion = "Core" + apiVersion
	}

	clientset, err := clientCtx.Clientset()
	if err != nil {
		return nil, err
	}
	clientSetValue := reflect.ValueOf(clientset)
	kindServiceMethod, ok := clientSetValue.Type().MethodByName(apiVersion)
	if !ok {
		return nil, fmt.Errorf("failed to locate api %v", apiVersion)
	}
	apiVersionValue := clientSetValue.Method(kindServiceMethod.Index).Interface()
	results := toolbox.CallFunction(apiVersionValue)

	apiVersionClient := reflect.ValueOf(results[0])
	kindServiceMethod, ok = apiVersionClient.Type().MethodByName(holder.name)
	if !ok {
		return nil, fmt.Errorf("failed to locate kind %v.%v", apiVersion, holder.name)
	}
	kindServiceValue := apiVersionClient.MethodByName(holder.name)
	if kindServiceMethod.Type.NumIn() == 1 {
		results = toolbox.CallFunction(kindServiceValue.Interface())
	} else {
		results = toolbox.CallFunction(kindServiceValue.Interface(), clientCtx.Namespace)
	}
	return results[0], nil
}

func actionName(holder *apiHolder, method reflect.Method) string {
	interfaceType := holder.iFace.String()
	kindName := kindName(interfaceType)
	action := method.Name + kindName
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

type apiHolder struct {
	name       string
	id         string
	impl       interface{}
	iFace      reflect.Type
	kind       string
	apiVersion string
}

func newServiceHolder(apiPrefix string, method reflect.Method) *apiHolder {
	apiPrefix = strings.Replace(apiPrefix, "core", "", 1)
	if apiPrefix != "" {
		apiPrefix += "/"
	}
	resultType := method.Type.Out(0)
	apiVersion := apiPrefix
	apiType := resultType.String()
	if index := strings.Index(apiType, "."); index != -1 {
		apiVersion += string(apiType[:index])
	}
	return &apiHolder{
		id:         apiPrefix + strings.Replace(resultType.String(), "Interface", "", 1),
		name:       method.Name,
		iFace:      resultType,
		kind:       util.SimpleTypeName(strings.Replace(resultType.String(), "Interface", "", 1)),
		apiVersion: apiVersion,
	}
}
