package shared

import (
	"fmt"
	"github.com/viant/toolbox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"
)

type KindOperations struct {
	metav1.TypeMeta
	Methods map[string]ContractAdapter
}

var converter = toolbox.NewConverter("", "json")

func (r *KindOperations) Lookup(method string) (ContractAdapter, error) {
	if method == "" {
		return nil, fmt.Errorf("method was empty")
	}
	adapter, ok := r.Methods[method]
	if !ok {
		return nil, fmt.Errorf("failed to lookup method: %v on %v.%v", method, r.APIVersion, r.Kind)
	}
	result := reflect.New(reflect.ValueOf(adapter).Elem().Type()).Interface()
	resultAdapter, ok := result.(ContractAdapter)
	if !ok {
		return nil, fmt.Errorf("unable to cast %T to ContractAdapter", result)
	}
	return resultAdapter, nil
}

func (r *KindOperations) NewRequest(method string, requestData interface{}) (interface{}, error) {
	request, err := r.Lookup(method)
	if err != nil {
		return nil, err
	}
	if err := converter.AssignConverted(request, requestData); err != nil {
		return nil, err
	}
	return request, nil
}

type KindMethodMeta struct {
	metav1.TypeMeta
	Method string
}

func NewKindMethodMeta(id string) *KindMethodMeta {
	fragments := strings.Split(id, ".")
	return &KindMethodMeta{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fragments[0],
			Kind:       strings.Replace(fragments[1], "Interface", "", 1),
		},
		Method: fragments[2],
	}
}
