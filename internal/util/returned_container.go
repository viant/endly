package util

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
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

// ExpandReturnedContainer restores native non-string variable normalization for
// data produced by a deferred expression, without mutating its retained source.
func ExpandReturnedContainer(state data.Map, value interface{}) interface{} {
	copyState := data.NewMap()
	value = copyState.Expand(value)
	if normalized, err := toolbox.NormalizeKVPairs(value); err == nil {
		value = normalized
	}
	return state.Expand(value)
}

// ExpandRequestContainers revisits only exact UDF leaves from the unexpanded
// request. Other already-expanded values retain their original evaluation count.
func ExpandRequestContainers(state data.Map, source, expanded interface{}) interface{} {
	if expression, ok := source.(string); ok {
		if IsSingleExpression(expression, true) && (toolbox.IsMap(expanded) || toolbox.IsSlice(expanded)) {
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
