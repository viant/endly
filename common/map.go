package common

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"strings"
)

type Map map[string]interface{}

func (s *Map) Put(key string, value interface{}) {
	(*s)[key] = value
}

func (s *Map) Delete(keys ...string) {
	for _, key := range keys {
		delete(*s, key)
	}
}

func (s *Map) Has(key string) bool {
	_, found := (*s)[key]
	return found
}

func (s *Map) Get(key string) interface{} {
	if result, found := (*s)[key]; found {
		return result
	}
	return nil

}

func (s *Map) GetValue(expr string) (interface{}, bool) {
	if expr == "" {
		return nil, false
	}
	isShiftOperation := strings.HasPrefix(expr, "<-")
	if isShiftOperation {
		expr = string(expr[2:])
	}

	isPostIncrementOperation := strings.HasSuffix(expr, "++")
	if isPostIncrementOperation {
		expr = string(expr[:len(expr)-2])
	}

	isPreIncrementOperation := strings.HasPrefix(expr, "++")
	if isPreIncrementOperation {
		expr = string(expr[2:])
	}

	isReference := strings.HasPrefix(expr, "$")
	if isReference {
		expr = string(expr[1:])
		expr = s.GetString(expr)
		if expr == "" {
			return nil, false
		}
	}

	state := *s
	if string(expr[0:1]) == "{" {
		expr = expr[1 : len(expr)-1]
	}
	if strings.Contains(expr, ".") {
		fragments := strings.Split(expr, ".")
		for i, fragment := range fragments {
			isLast := i+1 == len(fragments)
			hasKey := state.Has(fragment)
			if !hasKey {
				return nil, false
			}
			var candidate = state.Get(fragment)
			if !isLast &&  candidate == nil {
				return nil, false
			}
			if isLast {
				expr = fragment
			} else if toolbox.IsMap(candidate) {



				newState := state.GetMap(fragment)
				if newState != nil {
					state = newState
				}
			} else {

				value, _ := state.GetValue(fragment)
				if f, ok := value.(func(key string) interface{}); ok {
					return f(fragments[i+1]), true
				}
				return nil, false
			}
		}
	}

	if state.Has(expr) {
		var result = state.Get(expr)
		if isPostIncrementOperation {
			state.Put(expr, toolbox.AsInt(result)+1)
		} else if isPreIncrementOperation {
			result = toolbox.AsInt(result) + 1
			state.Put(expr, result)
		} else if isShiftOperation {

			aCollection := state.GetCollection(expr)
			if len(*aCollection) == 0 {
				return nil, false
			}
			var result = (*aCollection)[0]
			var newCollection = (*aCollection)[1:]
			state.Put(expr, &newCollection)
			return result, true
		}
		if f, ok := result.(func() interface{}); ok {
			return f(), true
		}
		return result, true
	}
	return nil, false
}

func (s *Map) SetValue(expr string, value interface{}) {
	if expr == "" {
		return
	}
	state := *s
	isReference := strings.HasPrefix(expr, "$")
	if isReference {
		expr = string(expr[1:])
		expr = s.GetString(expr)
		s.Put(expr, value)
	}

	isPushOperation := strings.HasPrefix(expr, "->")
	if isPushOperation {
		expr = string(expr[2:])
	}
	if string(expr[0:1]) == "{" {
		expr = expr[1 : len(expr)-1]
	}
	if strings.Contains(expr, ".") {
		fragments := strings.Split(expr, ".")
		for i, fragment := range fragments {
			isLast := i+1 == len(fragments)
			if isLast {
				expr = fragment
			} else {
				subState := state.GetMap(fragment)
				if subState == nil {
					subState = NewMap()
					state.Put(fragment, subState)
				}
				state = subState
			}
		}
	}

	if isPushOperation {
		collection := state.GetCollection(expr)
		if collection == nil {
			collection = NewCollection()
			state.Put(expr, collection)
		}
		collection.Push(value)
		state.Put(expr, collection)
		return
	}
	state.Put(expr, value)
}

func (s *Map) Apply(source map[string]interface{}) {
	for k, v := range source {
		(*s)[k] = v
	}
}

func (s *Map) GetString(key string) string {
	if result, found := (*s)[key]; found {
		return toolbox.AsString(result)
	}
	return ""
}

func (s *Map) GetCommandSession(key string) *ssh.MultiCommandSession {
	if result, found := (*s)[key]; found {
		if session, ok := result.(*ssh.MultiCommandSession); ok {
			return session
		}
	}
	return nil
}

func (s *Map) GetInt(key string) int {
	if result, found := (*s)[key]; found {
		return toolbox.AsInt(result)
	}
	return 0
}

func (s *Map) GetFloat(key string) float64 {
	if result, found := (*s)[key]; found {
		return toolbox.AsFloat(result)
	}
	return 0.0
}

func (s *Map) GetBoolean(key string) bool {
	if result, found := (*s)[key]; found {
		return toolbox.AsBoolean(result)
	}
	return false
}

func (s *Map) GetCollection(key string) *Collection {
	if result, found := (*s)[key]; found {
		collectionPoiner, ok := result.(*Collection)
		if ok {
			return collectionPoiner
		}
		aSlice, ok := result.([]interface{})
		collection := Collection(aSlice)
		if ok {
			return &collection
		}
	}
	return nil

}

func (s *Map) GetMap(key string) Map {
	if result, found := (*s)[key]; found {
		aMap, ok := result.(Map)
		if ok {
			return aMap
		}
		aMap, ok = result.(map[string]interface{})
		if ok {
			return aMap
		}
		var result =  toolbox.AsMap(result)
		(*s)[key] = result
		return result
	}
	return nil

}

func (s *Map) Clone() Map {
	var result = NewMap()
	for key, value := range *s {
		if aMap, casted := value.(Map); casted {
			result[key] = aMap.Clone()
		} else {
			result[key] = value
		}
	}
	return result
}

func NewMap() Map {
	return make(map[string]interface{})
}
