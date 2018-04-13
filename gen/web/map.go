package web

import (
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"strings"
)

type Map interface {
	Range(handler func(key string, value interface{}))
	Has(key string) bool
	Put(key string, value interface{})
	Get(key string) interface{}
	GetMap(key string) Map
	Remove(key string) Map
	SubMap(path string) Map
	Len() int
}

type mapSlice yaml.MapSlice

func (m *mapSlice) Put(key string, value interface{}) {
	for i, item := range *m {
		if item.Key == key {
			(*m)[i] = yaml.MapItem{key, value}
			return
		}
	}
	*m = append(*m, yaml.MapItem{key, value})
}

//Get returns a value for provided key
func (s *mapSlice) Range(handler func(key string, value interface{})) {
	for _, item := range *s {
		handler(toolbox.AsString(item.Key), item.Value)
	}
}

func (m *mapSlice) Get(key string) interface{} {
	for _, item := range *m {
		if item.Key == key {
			return item.Value
		}
	}
	return nil
}

func (m *mapSlice) Has(key string) bool {
	for _, item := range *m {
		if item.Key == key {
			return true
		}
	}
	return false
}

func (m *mapSlice) Len() int {
	return len(*m)
}

func (m *mapSlice) Remove(key string) Map {
	var result = NewMap()
	for _, item := range *m {
		if item.Key == key {
			continue
		}
		result.Put(toolbox.AsString(item.Key), item.Value)
	}
	return result
}

func (m *mapSlice) SubMap(path string) Map {
	var result Map = m
	var keys = strings.Split(path, ".")
	for _, key := range keys {
		result = result.GetMap(key)
	}
	return result
}

func (m *mapSlice) AsYAML() yaml.MapSlice {
	var result = yaml.MapSlice(*m)
	for i, item := range result {
		if aMap, ok := item.Value.(Map); ok {
			result[i] = yaml.MapItem{item.Key, aMap}
		}
	}
	return result
}

func (m *mapSlice) GetMap(key string) Map {
	var value = m.Get(key)
	if value == nil {
		var value = NewMap()
		m.Put(key, value)
		return value
	}
	switch actual := value.(type) {
	case []yaml.MapItem:
		var aMapSlice = mapSlice(actual)
		var aMap = Map(&aMapSlice)
		m.Put(key, aMap)
		return aMap
	case yaml.MapSlice:
		var aMapSlice = mapSlice(actual)
		var aMap = Map(&aMapSlice)
		m.Put(key, aMap)
		return aMap

	case Map:
		return actual
	}
	return nil
}

func NewMap() Map {
	return &mapSlice{}
}
