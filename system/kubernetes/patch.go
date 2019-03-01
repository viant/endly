package core

import (
	"encoding/json"
	"github.com/mattbaird/jsonpatch"
	"github.com/viant/toolbox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//ResourcePatch represents resource patch
type ResourcePatch struct {
	metav1.TypeMeta `json:",inline"`
	Name            string
	Pt              types.PatchType
	Data            []byte
	HasChanged      bool
}

var patchableKeys = []string{"spec", "metadata"}

//NewResourcePatch returns a new resource patch
func NewResourcePatch(meta *ResourceMeta, original, target interface{}) (*ResourcePatch, error) {
	patch := &ResourcePatch{
		TypeMeta: meta.TypeMeta,
		Name:     meta.Metadata.Name,
	}
	var err error
	if original, err = filterFields(original, patchableKeys); err != nil {
		return nil, err
	}
	if target, err = filterFields(target, patchableKeys); err != nil {
		return nil, err
	}
	originalSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	targetJSON, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	patchOperations, err := jsonpatch.CreatePatch(originalSON, targetJSON)
	if err != nil {
		return nil, err
	}
	patchOperations = deleteRemoval(patchOperations)
	patch.HasChanged = len(patchOperations) > 0
	if patch.Data, err = json.Marshal(patchOperations); err != nil {
		return nil, err
	}
	patch.Pt = types.JSONPatchType
	return patch, nil
}

func filterFields(resource interface{}, keys []string) (interface{}, error) {
	var original = make(map[string]interface{})
	err := converter.AssignConverted(&original, resource)
	if err != nil {
		return nil, err
	}
	original = toolbox.DeleteEmptyKeys(original)
	var targetMap = make(map[string]interface{})
	for _, key := range keys {
		targetMap[key] = original[key]
	}
	return targetMap, err
}

func deleteRemoval(operations []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	var result = make([]jsonpatch.JsonPatchOperation, 0)
	for _, operation := range operations {
		if operation.Operation == "remove" {
			continue
		}
		if operation.Operation == "replace" {
			if operation.Value == nil || (toolbox.IsNumber(operation.Value) && toolbox.AsInt(operation.Value) == 0) {
				continue
			}
		}
		result = append(result, operation)
	}
	return result
}
