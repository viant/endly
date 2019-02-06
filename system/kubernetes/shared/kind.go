package shared

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type KindOperations struct {
	metav1.TypeMeta
	Methods map[string]ContractAdapter
}

func (r *KindOperations) Lookup(method string) (ContractAdapter, error) {
	result, ok := r.Methods[method]
	if !ok {
		return nil, fmt.Errorf("failed to lookup method: %v on %v.%v", method, r.APIVersion, r.Kind)
	}
	return result, nil
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
