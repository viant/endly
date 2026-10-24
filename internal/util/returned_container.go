package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"reflect"
)

// IsSingleExpression uses native expression boundaries, excluding interpolation.
func IsSingleExpression(source string, udf bool) bool {
	const marker = 987654321
	parsed := data.Parse(source, func(_ string, isUDF bool, _ interface{}) (interface{}, bool) {
		if isUDF != udf {
			return nil, false
		}
		return marker, true
	})
	value, ok := parsed.(int)
	return ok && value == marker
}

// IsContainer excludes nil and typed-nil values before native reflection helpers.
func IsContainer(value interface{}) bool {
	return !isNilValue(value) && (toolbox.IsMap(value) || toolbox.IsSlice(value))
}

func isNilValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflected := reflect.ValueOf(value); reflected.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan:
		return reflected.IsNil()
	}
	return false
}

// ExpandReturnedContainer restores native non-string variable normalization for
// data produced by a deferred expression, without mutating its retained source.
func ExpandReturnedContainer(state data.Map, value interface{}) interface{} {
	if !IsContainer(value) {
		return value
	}
	copyState := data.NewMap()
	value = copyState.Expand(value)
	if normalized, err := normalizeKVPairs(value); err == nil {
		value = normalized
	}
	return state.Expand(value)
}

// ExpandRequestContainers revisits only exact UDF leaves from the unexpanded
// request. Other already-expanded values retain their original evaluation count.
func ExpandRequestContainers(state data.Map, source, expanded interface{}) interface{} {
	if isNilValue(source) || isNilValue(expanded) {
		return expanded
	}
	if expression, ok := source.(string); ok {
		if IsSingleExpression(expression, true) && IsContainer(expanded) {
			return ExpandReturnedContainer(state, expanded)
		}
		return expanded
	}
	if toolbox.IsMap(source) && toolbox.IsMap(expanded) {
		original := toolbox.AsMap(source)
		result := toolbox.AsMap(expanded)
		for key, value := range original {
			if actual, ok := result[key]; ok {
				result[key] = ExpandRequestContainers(state, value, actual)
			}
		}
		return result
	}
	if toolbox.IsSlice(source) && toolbox.IsSlice(expanded) {
		original := toolbox.AsSlice(source)
		result := toolbox.AsSlice(expanded)
		for i, value := range original {
			if i < len(result) {
				result[i] = ExpandRequestContainers(state, value, result[i])
			}
		}
		return result
	}
	return expanded
}
