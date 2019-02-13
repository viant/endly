package shared

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
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
	if kind == "" {
		return nil, fmt.Errorf("kind was empty")
	}

	if strings.Contains(kind, ".") && apiVersion == "" {
		pair := strings.SplitN(kind, ".", 2)
		apiVersion = pair[0]
		kind = pair[1]
	}

	if apiVersion == "" {
		apiVersion, _ = LookupAPIVersion(kind)
	}

	if apiVersion == "" {
		return nil, fmt.Errorf("apiVersion was empty")
	}

	byKind, ok := typeMetaRegistry[apiVersion]
	if !ok {
		available := toolbox.MapKeysToStringSlice(typeMetaRegistry)
		return nil, fmt.Errorf("failed to lookup api %v, avaiale: %v", apiVersion, available)

	}
	result, ok := byKind[strings.Title(kind)]
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
	_, has := defaultKindAPIVersion[meta.Kind]
	if !has {
		defaultKindAPIVersion[meta.Kind] = []string{meta.APIVersion}
	} else {
		for _, api := range defaultKindAPIVersion[meta.Kind] {
			if api == meta.APIVersion {
				return
			}
		}
		defaultKindAPIVersion[meta.Kind] = append(defaultKindAPIVersion[meta.Kind], meta.APIVersion)
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

//MetaTypes returns meta types
func MetaTypes() []string {
	var index = make(map[string]bool)
	var result = make([]string, 0)
	for _, api := range typeMetaRegistry {
		for _, kind := range api {
			if _, has := index[kind.Kind]; has {
				continue
			}
			index[kind.Kind] = true
			result = append(result, kind.APIVersion+"."+kind.Kind)
		}
	}
	return result
}

//MatchedMetaTypes returns matched kinds
func MatchedMetaTypes(kinds ...string) []string {
	var index = make(map[string]bool)
	for _, kind := range kinds {
		index[kind] = true
	}
	var result = make([]string, 0)
	for _, api := range typeMetaRegistry {
		for _, kind := range api {
			if _, has := index[kind.Kind]; !has {
				continue
			}
			result = append(result, kind.APIVersion+"."+kind.Kind)
		}
	}
	return result
}
