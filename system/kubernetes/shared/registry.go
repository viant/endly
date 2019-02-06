package shared

import (
	"fmt"
	"github.com/viant/toolbox"
	"sort"
)

var registry = make(map[string]ContractAdapter, 0)
var typeMetaRegistry byAPI = make(map[string]byKind)

//Register register an adapter
func Register(adapter ContractAdapter) {
	id := adapter.GetId()
	registry[id] = adapter
	meta := NewKindMethodMeta(id)
	typeMetaRegistry.Put(meta, adapter)
}

//Get returns contract adapter
func Get(id string) (ContractAdapter, bool) {
	result, ok := registry[id]
	return result, ok
}

//Lookup returns KindOperations with all defined operations or error
func Lookup(apiVersion, kind string) (*KindOperations, error) {
	if apiVersion == "" {
		return nil, fmt.Errorf("apiVersion was empty")
	}
	if apiVersion == "" {
		return nil, fmt.Errorf("kind was empty")
	}
	byKind, ok := typeMetaRegistry[apiVersion]
	if !ok {
		available := toolbox.MapKeysToStringSlice(typeMetaRegistry)
		return nil, fmt.Errorf("failed to lookup api %v, avaiale: %v", apiVersion, available)

	}
	result, ok := byKind[kind]
	if !ok {
		return nil, fmt.Errorf("failed to lookup kind %v in api: %v", kind, apiVersion)
	}
	return result, nil
}

type byKind map[string]*KindOperations
type byAPI map[string]byKind

func (by *byAPI) Put(meta *KindMethodMeta, adapter ContractAdapter) {
	if _, has := (*by)[meta.APIVersion]; !has {
		(*by)[meta.APIVersion] = make(map[string]*KindOperations)
	}
	byKind := (*by)[meta.APIVersion]
	byKind.Put(meta, adapter)
	version, has := defaultKindAPIVersion[meta.Kind]
	if !has {
		defaultKindAPIVersion[meta.Kind] = meta.APIVersion
	} else if version != meta.APIVersion {
		sortedVersion := []string{version, meta.APIVersion}
		sort.Strings(sortedVersion)
		defaultKindAPIVersion[meta.Kind] = sortedVersion[0]
	}
}

func (by *byKind) Put(meta *KindMethodMeta, adapter ContractAdapter) {
	if _, has := (*by)[meta.Kind]; !has {
		(*by)[meta.Kind] = &KindOperations{
			TypeMeta: meta.TypeMeta,
			Methods:  make(map[string]ContractAdapter),
		}
	}
	(*by)[meta.Kind].Methods[meta.Method] = adapter
}
