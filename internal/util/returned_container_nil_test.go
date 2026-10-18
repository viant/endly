package util

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestRequestContainerExpansionNilValues(t *testing.T) {
	var nilMap map[string]interface{}
	var nilList []interface{}
	var nilPointer *map[string]interface{}
	for _, tc := range []struct {
		name             string
		source, expanded interface{}
	}{
		{"both nil", nil, nil},
		{"nil source", nil, map[string]interface{}{"x": nil}},
		{"nil expanded", map[string]interface{}{"x": 1}, nil},
		{"nil map", nilMap, nilMap},
		{"nil list", nilList, nilList},
		{"nil pointer", nilPointer, nilPointer},
		{"UDF nil", "$Produced()", nil},
		{"UDF typed nil map", "$Produced()", nilMap},
		{"UDF typed nil list", "$Produced()", nilList},
		{"null map member", map[string]interface{}{"x": nil}, map[string]interface{}{"x": nil}},
		{"null list member", []interface{}{nil}, []interface{}{nil}},
		{"nested nulls", map[string]interface{}{"list": []interface{}{nil, map[string]interface{}{"x": nil}}}, map[string]interface{}{"list": []interface{}{nil, map[string]interface{}{"x": nil}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				got := ExpandRequestContainers(data.NewMap(), tc.source, tc.expanded)
				require.Equal(t, tc.expanded, got)
			})
		})
	}
}
func TestReturnedContainerNilPreservation(t *testing.T) {
	var nilMap map[string]interface{}
	var nilList []interface{}
	for _, value := range []interface{}{nil, nilMap, nilList} {
		require.NotPanics(t, func() { require.Equal(t, value, ExpandReturnedContainer(data.NewMap(), value)) })
	}
}
