package core

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

//GenerateRequest converts request with provided template and call handler with meta and request data.
func GenerateRequest(name string, templates map[string]string, templateParams interface{}, handler func(meta *ResourceMeta, data map[string]interface{}) error) error {
	var templateParamMap = make(map[string]interface{})
	err := converter.AssignConverted(&templateParamMap, templateParams)
	if err != nil {
		return err
	}
	templateData := data.Map(templateParamMap)
	var spec map[string]interface{}
	if spec, err = loadTemplate(podSpec); err != nil {
		return err
	}

	templateData["Spec"] = toolbox.AsMap(templateData.Expand(spec))
	template, ok := templates[name]
	if !ok {
		return fmt.Errorf("failed to lookup template: %v", name)
	}
	templateMap, err := loadTemplate(template)
	if err != nil {
		return err
	}

	requestData := toolbox.AsMap(templateData.Expand(templateMap))
	meta := &ResourceMeta{}
	if err = converter.AssignConverted(meta, requestData); err != nil {
		return err
	}
	requestData = toolbox.DeleteEmptyKeys(requestData)
	return handler(meta, requestData)
}

func loadTemplate(template string) (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(template), &result); err != nil {
		return nil, err
	}
	normalized, err := toolbox.NormalizeKVPairs(result)
	if err != nil {
		return nil, err
	}
	result = toolbox.AsMap(normalized)
	return result, nil
}

func buildResourceRequirements(requests, limits map[string]string) (*v1.ResourceRequirements, error) {
	hasRequests := len(requests) > 0
	hasLimits := len(limits) > 0
	if !hasRequests && !hasLimits {
		return nil, nil
	}
	result := &v1.ResourceRequirements{
		Limits:   make(map[v1.ResourceName]resource.Quantity),
		Requests: make(map[v1.ResourceName]resource.Quantity),
	}
	if hasLimits {
		if err := applyResourceList(limits, result.Limits, "limits"); err != nil {
			return nil, err
		}
	}
	if hasLimits {
		if err := applyResourceList(requests, result.Requests, "requests"); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func applyResourceList(values map[string]string, resourceList v1.ResourceList, listType string) error {
	for k, v := range values {
		qty, err := resource.ParseQuantity(v)
		if err != nil {
			return fmt.Errorf("unable to parse %v %v: %v, %v", listType, k, v, err)
		}
		resourceList[v1.ResourceName(k)] = qty
	}
	return nil
}
