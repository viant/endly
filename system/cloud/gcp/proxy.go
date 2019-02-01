package gcp

import (
	"encoding/json"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"reflect"
	"strings"
)

type EmptyOutput struct{}

func extractServices(owner interface{}, serviceFields *[]*toolbox.StructField) error {
	return toolbox.ProcessStruct(owner, func(fieldType reflect.StructField, field reflect.Value) error {
		if fieldType.PkgPath != "" {
			return nil
		}
		if !toolbox.IsStruct(fieldType.Type) {
			return nil
		}
		*serviceFields = append(*serviceFields, &toolbox.StructField{Type: fieldType, Value: field})

		if field.Kind() != reflect.Ptr {
			return nil
		}
		serviceCandidate := reflect.New(field.Type().Elem()).Elem().Interface()
		return extractServices(serviceCandidate, serviceFields)
	})

}

func BuildRoutes(service interface{}, nameTransformer func(name string) string, clientProvider func(context *endly.Context) (CtxClient, error)) ([]*endly.Route, error) {
	var fields = make([]*toolbox.StructField, 0)
	err := extractServices(service, &fields)
	if err != nil {
		return nil, err
	}
	var result = make([]*endly.Route, 0)
	for _, structField := range fields {
		fieldType := structField.Type
		service := reflect.New(fieldType.Type.Elem()).Interface()
		if err = toolbox.ScanStructMethods(service, 1, func(method reflect.Method) error {
			outNum := method.Type.NumOut()
			if outNum != 1 {
				return nil
			}
			requestType := method.Type.Out(0)
			doMethod, hasDo := requestType.MethodByName("Do")
			contextMethod, hasContext := requestType.MethodByName("Context")
			if !hasDo || !hasContext {
				return nil
			}
			responseType := doMethod.Func.Type().Out(0)
			if responseType.Kind() == reflect.Interface {
				responseType = reflect.TypeOf(&EmptyOutput{})
			}

			action := fieldType.Name + method.Name
			if nameTransformer != nil {
				action = nameTransformer(action)
			}
			action = normalizeAction(action)

			route := &endly.Route{
				Action: action,
				RequestInfo: &endly.ActionInfo{
					Description: fmt.Sprintf("%T.%v(%T)", service, method.Name, reflect.New(requestType.Elem()).Interface()),
				},
				ResponseInfo: &endly.ActionInfo{
					Description: fmt.Sprintf("%T", reflect.New(responseType.Elem()).Interface()),
				},

				RequestProvider: func() interface{} {
					return reflect.New(requestType.Elem()).Interface()
				},

				ResponseProvider: func() interface{} {
					return reflect.New(responseType.Elem()).Interface()
				},
				Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
					client, err := clientProvider(context)
					if err != nil {
						return nil, err
					}
					_ = toolbox.CallFunction(contextMethod.Func.Interface(), request, client.Context())
					output := toolbox.CallFunction(doMethod.Func.Interface(), request)
					if len(output) > 1 {
						errOutput := output[1]
						if errOutput != nil {
							return nil, fmt.Errorf("unable to run %v, %v", action, errOutput)
						}
					}
					var result interface{}
					if len(output) > 0 {
						result = output[0]
					}
					if context.IsLoggingEnabled() {
						aMap := make(map[string]interface{})
						if toolbox.DefaultConverter.AssignConverted(&aMap, result) == nil {
							result = normalizeMap(aMap)
						}
						context.Publish(NewOutputEvent(method.Name, "proxy", result))
					}
					return result, nil
				},
			}
			result = append(result, route)
			return nil
		}); err != nil {
			return nil, err
		}
	}
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

func normalizeMap(dataMap map[string]interface{}) map[string]interface{} {
	if len(dataMap) == 0 {
		return dataMap
	}
	for k, v := range dataMap {
		if v != nil && toolbox.IsMap(v) {
			dataMap[k] = normalizeMap(toolbox.AsMap(v))
			continue
		}
		if toolbox.IsSlice(v) {
			aSlice := toolbox.AsSlice(v)
			if len(aSlice) == 0 {
				continue
			}
			if _, ok := aSlice[0].(byte); ok {
				rawData := toolbox.AsString(v)
				aMap := make(map[string]interface{})
				if err := json.Unmarshal([]byte(rawData), &aMap); err == nil {
					dataMap[k] = aMap
				}
			}
			for i, item := range aSlice {
				if !toolbox.IsMap(item) {
					break
				}
				aSlice[i] = normalizeMap(toolbox.AsMap(item))
			}
		}
	}
	return dataMap
}
