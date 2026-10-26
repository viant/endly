package endly_test

import (
	"reflect"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

func TestIsDefinedNestedPresence(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"empty expression", "", false},
		{"missing root", "missing", false},
		{"present scalar", "scalar", true},
		{"present nil", "nilLeaf", true},
		{"nil intermediate", "nilLeaf.child", false},
		{"two levels", "response.StructuredContent", true},
		{"nonempty data", "response.StructuredContent.data", true},
		{"empty data", "response.StructuredContent.empty", true},
		{"nil data", "response.StructuredContent.nil", true},
		{"false data", "response.StructuredContent.false", true},
		{"zero data", "response.StructuredContent.zero", true},
		{"missing data", "response.StructuredContent.missing", false},
		{"deep map", "response.StructuredContent.nested.value", true},
		{"missing deep map", "response.StructuredContent.nested.missing", false},
		{"scalar intermediate", "scalar.child", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := endly.NewDefaultState(nil)
			state["scalar"] = "unchanged"
			state["nilLeaf"] = nil
			body := map[string]interface{}{
				"data": []interface{}{1}, "empty": []interface{}{}, "nil": nil,
				"false": false, "zero": 0, "nested": data.Map{"value": "kept"},
			}
			response := map[string]interface{}{"StructuredContent": body}
			state["response"] = response
			// Assert the evaluated boolean directly, before any assertly comparison.
			got := state.Expand("$IsDefined(" + test.path + ")")
			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("IsDefined(%q) = %#v (%T), want %v", test.path, got, got, test.want)
			}
			if state["scalar"] != "unchanged" || response["StructuredContent"] == nil ||
				!reflect.DeepEqual(body["data"], []interface{}{1}) ||
				!reflect.DeepEqual(body["empty"], []interface{}{}) || body["nil"] != nil {
				t.Fatal("presence lookup mutated response or leaf values")
			}
		})
	}
}

func TestIsDefinedDoesNotReadRootCollision(t *testing.T) {
	state := endly.NewDefaultState(nil)
	state["response"] = map[string]interface{}{"StructuredContent": map[string]interface{}{}}
	state["StructuredContent"] = map[string]interface{}{"data": "unrelated root value"}
	if got := state.Expand("$IsDefined(response.StructuredContent.data)"); got != false {
		t.Fatalf("missing nested data resolved through unrelated root: %#v", got)
	}
}

func TestIsDefinedPreservesTypedResponse(t *testing.T) {
	type response struct {
		StructuredContent map[string]interface{}
	}
	value := &response{StructuredContent: map[string]interface{}{"data": []interface{}{1}}}
	state := endly.NewDefaultState(nil)
	state["response"] = value
	if got := state.Expand("$IsDefined(response.StructuredContent.data)"); got != true {
		t.Fatalf("typed response presence = %#v, want true", got)
	}
	if state["response"] != value {
		t.Fatal("presence lookup replaced the typed response")
	}
}
