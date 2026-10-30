package udf

import (
	"encoding/json"
	"testing"

	"github.com/viant/toolbox/data"
)

func TestAsMapJSONPreservesNumbersAndContainers(t *testing.T) {
	const source = `{"large":9007199254740993,"unsigned":18446744073709551615,"fraction":1.25,"zero":0,"trial":false,"nullable":null,"list":[],"object":{},"pairs":[{"Key":"name","Value":"value"}]}`
	for _, input := range []interface{}{source, []byte(source)} {
		actual, err := AsMap(input, data.NewMap())
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(actual)
		if err != nil {
			t.Fatal(err)
		}
		// Decode again using the same JSON semantics to compare canonical objects.
		var expected map[string]json.RawMessage
		if err := json.Unmarshal([]byte(source), &expected); err != nil {
			t.Fatal(err)
		}
		want, _ := json.Marshal(expected)
		if string(encoded) != string(want) {
			t.Fatalf("JSON changed: want %s got %s", want, encoded)
		}
	}
}

func TestAsMapRetainsNativeInputs(t *testing.T) {
	for _, input := range []interface{}{nil, map[string]interface{}{"name": "value"}, "name: value"} {
		actual, err := AsMap(input, data.NewMap())
		if err != nil {
			t.Fatal(err)
		}
		if input == nil {
			if actual != nil {
				t.Fatalf("nil became %#v", actual)
			}
			continue
		}
		if actual.(map[string]interface{})["name"] != "value" {
			t.Fatalf("native input changed: %#v", actual)
		}
	}
}

func TestAsMapRejectsInvalidJSONAndNumericOverflow(t *testing.T) {
	for _, source := range []string{`{"broken":}`, `{"value":1} {"extra":2}`, `{"overflow":18446744073709551616}`} {
		if _, err := AsMap(source, data.NewMap()); err == nil {
			t.Fatalf("expected conversion error for %s", source)
		}
	}
}
