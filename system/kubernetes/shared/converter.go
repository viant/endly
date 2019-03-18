package shared

import (
	"fmt"
	"github.com/viant/toolbox"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"reflect"
)

var iFace interface{}
var inerfacePtrType = reflect.TypeOf(&iFace)

var quantityPtrType = reflect.TypeOf(&resource.Quantity{})
var stringType = reflect.TypeOf("")
var resourceListType = reflect.TypeOf(v1.ResourceList{})

func TextToQuantity(target, source interface{}) error {
	dest, ok := target.(*resource.Quantity)
	if !ok {
		return fmt.Errorf("expected %T, but had: %T", dest, target)
	}
	src, ok := source.(string)
	if !ok {
		return fmt.Errorf("expected %T, but had: %T", src, source)
	}
	srcQuantity, err := resource.ParseQuantity(src)
	if err != nil {
		return err
	}
	*dest = srcQuantity
	return nil
}

func ResourceListToInterface(target, source interface{}) error {
	dest, ok := target.(*interface{})
	if !ok {
		return fmt.Errorf("expected %T, but had: %T", dest, target)
	}
	src, ok := source.(v1.ResourceList)
	if !ok {
		return fmt.Errorf("expected %T, but had: %T", src, source)
	}
	var result = make(map[string]interface{})
	for k, v := range src {
		result[string(k)] = v.String()
	}
	*dest = result
	return nil
}

func init() {
	toolbox.RegisterConverter(quantityPtrType, stringType, TextToQuantity)
	toolbox.RegisterConverter(inerfacePtrType, resourceListType, ResourceListToInterface)

}
