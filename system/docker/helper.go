package docker

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
	"os"
	"reflect"
	"strings"
)

func IsContainerUp(container *types.Container) bool {
	if container == nil {
		return false
	}
	return strings.Contains(strings.ToLower(container.Status), "up")
}

// runAdapter runs adapter request
func runAdapter(context *endly.Context, adapter ContractAdapter, response interface{}) error {
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return err
	}
	adapter.SetContext(ctxClient.Context)
	if err = adapter.SetService(ctxClient.Client); err != nil {
		return err
	}
	resp, err := adapter.Call()
	if err != nil {
		return err
	}
	if response == nil {
		return nil
	}
	responseValue := reflect.ValueOf(response)
	if responseValue.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid response type: expected %T, but had %T", response, resp)
	}
	responseValue.Elem().Set(reflect.ValueOf(resp))
	return nil
}

func publishEvent(context *endly.Context, method string, value interface{}) {
	if value == nil || !context.IsLoggingEnabled() {
		return
	}

	eventValue := value
	if toolbox.IsSlice(value) {
		var aSlice = make([]interface{}, 0)
		if err := toolbox.DefaultConverter.AssignConverted(&aSlice, value); err == nil {
			eventValue = aSlice
		}
	} else {
		var aMap = make(map[string]interface{})
		if err := toolbox.DefaultConverter.AssignConverted(&aMap, value); err == nil {
			aMap = toolbox.DeleteEmptyKeys(aMap)
			eventValue = aMap
		}
	}
	context.Publish(&OutputEvent{msg.NewOutputEvent(method, "docker", eventValue)})
}

func expandHomeDirectory(location string) string {
	if strings.HasPrefix(location, "~") {
		location = strings.Replace(location, "~", os.Getenv("HOME"), 1)
	}
	return location
}
