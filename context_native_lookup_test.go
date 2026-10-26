package endly_test

import (
	"github.com/viant/endly"
	"testing"
)

func TestIsDefinedPublicTypedPathsArePassive(t *testing.T) {
	type record struct {
		Value interface{} `json:"value"`
	}
	type response struct {
		Entries []*record `json:"entries"`
	}
	state := endly.NewDefaultState(nil)
	original := &response{Entries: []*record{{Value: nil}}}
	state["response"] = original
	state["counter"] = 1
	calls := 0
	state["getter"] = func(string) interface{} { calls++; return "not-called" }
	for expression, want := range map[string]bool{
		"$IsDefined(response.entries[0].value)":   true,
		"$IsDefined(response.entries[0].missing)": false,
		"$IsDefined(response.entries[-1].value)":  false,
		"$IsDefined(response.entries[1].value)":   false,
		"$IsDefined(counter++)":                   false,
		"$IsDefined(getter.field)":                false,
	} {
		if value := state.Expand(expression); value != want {
			t.Fatalf("%s=%#v (%T), want %t", expression, value, value, want)
		}
	}
	if calls != 0 || state["counter"] != 1 || state["response"] != original {
		t.Fatal("presence lookup mutated or evaluated caller state")
	}
}
