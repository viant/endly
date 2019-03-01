package core

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/data/udf"
	"strings"
)

var outputTemplates = map[string]map[string]string{
	"pod": {
		"namespace": "$namespace",
		"name":      "$name",
		"kind":      "$kind",
		"status":    "$status.phase",
		"age":       "$Elapsed('$status.startTime')",
		"ip":        "$status.podIP",
		"hostIP":    "$status.hostIP",
		"labels":    "$labels",
		"ports":     "$Select('spec/containers/*/ports/*','containerPort:port', 'protocol:proto'",
		"volumes":   "$Select('spec/containers/*/volumeMounts/*','name', 'mountPath:mount')",
	},
	"service": {
		"namespace":  "$namespace",
		"name":       "$name",
		"kind":       "$kind",
		"type":       "$spec.type",
		"clusterIP":  "$spec.clusterIP",
		"externalIP": "$spec.externalIPs",
		"labels":     "$meta.labels",
		"selector":   "$selector",
		"ports":      "$spec.ports",
	},
	"replicaset": {
		"namespace":  "$namespace",
		"name":       "$name",
		"kind":       "$kind",
		"age":        "$Elapsed('$creationTimestamp')",
		"desired":    "$spec.replicas",
		"current":    "$status.replicas",
		"selector":   "$spec.selector",
		"containers": "$Select('spec/template/spec/containers/*','name','image','ports')",
	},
	"deployment": {
		"uid":        "$uid",
		"namespace":  "$namespace",
		"name":       "$name",
		"kind":       "$kind",
		"age":        "$Elapsed('$creationTimestamp')",
		"status":     "$status.conditions",
		"labels":     "$spec.template.labels",
		"replicas":   "$spec.replicas",
		"containers": "$Select('spec/template/spec/containers/*','name','image','ports')",
	},
	"endpoints": {
		"namespace": "$namespace",
		"name":      "$name",
		"kind":      "$kind",
		"age":       "$Elapsed('$creationTimestamp')",
		"subsets":   "$subsets",
		"labels":    "$labels",
	},
	"node": {
		"name":     "$name",
		"kind":     "$kind",
		"labels":   "$labels",
		"nodeInfo": "$status.nodeInfo",
	},
}

func transformOutput(outputPaths map[string]string, source, target map[string]interface{}) {
	sourceMap := data.Map(source)
	udf.Register(sourceMap)
	targetMap := data.Map(target)

	for k, v := range outputPaths {
		value := sourceMap.Expand(v)
		if value == nil || toolbox.AsString(value) == "" || value == v {
			continue
		}
		targetMap.SetValue(k, value)
	}
}

func getOutputRules(kind string) map[string]string {
	return outputTemplates[strings.ToLower(kind)]
}

//ShortResourceResponse returns short info
func ShortItemInfo(kind string, item map[string]interface{}) map[string]interface{} {
	rules := getOutputRules(kind)
	if len(rules) == 0 {
		return item
	}
	var result = make(map[string]interface{})
	transformOutput(rules, item, result)
	return result
}

//ShortResourceResponse represents resource meta info
type ShortResourceResponse struct {
	Items []interface{}
}

//NewShortResourceResponse create short response
func NewShortResourceResponse(response ResourceInfoResponse) *ShortResourceResponse {
	result := &ShortResourceResponse{
		Items: make([]interface{}, 0),
	}
	for _, item := range response.Items {
		rawMap := toolbox.AsMap(item.Raw)
		rawMap["kind"] = item.Kind
		rawMap["apiVersion"] = item.APIVersion
		result.Items = append(result.Items, ShortItemInfo(item.Kind, toolbox.AsMap(item.Raw)))
	}
	return result
}

func normalizeOutput(response interface{}) interface{} {
	var aMap = make(map[string]interface{})
	if err := converter.AssignConverted(&aMap, response); err != nil {
		return err
	}
	aMap = toolbox.DeleteEmptyKeys(aMap)
	if items, ok := aMap["items"]; ok {
		if len(toolbox.AsSlice(items)) > 1 {
			delete(aMap, "metadata")
		}
	}
	return aMap
}
