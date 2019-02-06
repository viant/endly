package core

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/data/udf"
	"strings"
)

const defaultOutputTemplate = "default"

var outputTemplates = map[string]map[string]string{
	"pod": {
		"namespace": "$namespace",
		"name":      "$name",
		"status":    "$status.phase",
		"age":       "$Elapsed('$status.startTime')",
		"ip":        "$status.podIP",
	},
	"service": {
		"namespace":  "$namespace",
		"name":       "$name",
		"type":       "$spec.type",
		"clusterIP":  "$spec.clusterIP",
		"externalIP": "$spec.externalIPs",
		"selector":   "$spec.selector",
		"ports":      "$spec.ports",
	},
	"replicaset": {
		"namespace":  "$namespace",
		"name":       "$name",
		"desired":    "$spec.replicas",
		"current":    "$status.replicas",
		"selector":   "$spec.selector",
		"containers": "$spec.template.spec.containers",
	},
}

func transformOutput(outputPaths map[string]string, source, target map[string]interface{}) {
	sourceMap := data.Map(source)
	udf.Register(sourceMap)
	for k, v := range outputPaths {
		value := sourceMap.Expand(v)
		if value == nil || toolbox.AsString(value) == "" || value == v {
			continue
		}
		target[k] = value
	}
}

func getOutputRules(template, kind string, outputPaths map[string]string) map[string]string {
	if len(outputPaths) > 0 {
		return outputPaths
	}
	outputPaths, ok := outputTemplates[template]
	if !ok {
		outputPaths, _ = outputTemplates[strings.ToLower(kind)]
	}
	return outputPaths
}
