package compiler_test

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria/compiler"
	"github.com/viant/toolbox/data"
	"testing"
)

func nativeCriterion(t *testing.T, state data.Map, expression string) (interface{}, bool, error) {
	t.Helper()
	build, err := compiler.Compile(expression)
	if err != nil {
		t.Fatal(err)
	}
	evaluate, err := build()
	if err != nil {
		t.Fatal(err)
	}
	return evaluate(state)
}

func TestNativeNumericComparisonRejectsUnresolvedUDF(t *testing.T) {
	state := endly.NewDefaultState(nil)
	for _, expression := range []string{
		"$UnknownUDF() >= 0", "$Len($missing) >= 0", "$IndexOf($missing, alpha) >= 0",
		"$UnknownUDF() < 1", "$UnknownUDF() <= 0", "$UnknownUDF() > -1",
	} {
		value, _, err := nativeCriterion(t, state, expression)
		t.Logf("%s -> %#v (%T), error=%v", expression, value, value, err)
		if err == nil {
			t.Fatalf("unresolved UDF was silently compared numerically: %s -> %#v", expression, value)
		}
	}
}

func TestNativeNumericComparisonPreservesOptionalAndLiteralValues(t *testing.T) {
	state := endly.NewDefaultState(nil)
	state["zero"], state["false"], state["nil"], state["optional"] = 0, false, nil, nil
	state["literal"] = "$ForeignDSL(keep)"
	for expression, want := range map[string]bool{
		"0 >= 0": true, "-1 >= 0": false, "$zero >= 0": true, "$false <= 0": true,
		"$nil >= 0": true, "$nil == nil": false, "$nil == $nil": true, "$Len($optional) = 0": true,
		"!$IsDefined(missing) || $Len($missing) = 0": true,
		"$literal == '$ForeignDSL(keep)'":            true,
	} {
		value, _, err := nativeCriterion(t, state, expression)
		if err != nil || value != want {
			t.Fatalf("%s -> %#v (%T), error=%v; want %t", expression, value, value, err, want)
		}
	}
	if state["literal"] != "$ForeignDSL(keep)" || state["false"] != false || state["zero"] != 0 || state["nil"] != nil {
		t.Fatal("criterion mutated literal/state values")
	}
}

func TestNativeDynamicReaderSelectionEvaluatesExactlyTwo(t *testing.T) {
	state := endly.NewDefaultState(nil)
	state["choices"] = []string{"reader03", "reader47"}
	readers := make([]map[string]interface{}, 52)
	for i := range readers {
		readers[i] = map[string]interface{}{"name": fmt.Sprintf("reader%02d", i)}
	}
	state["readers"] = readers
	selected := []string{}
	for i := range readers {
		state["index"] = i
		raw := state.Expand("$IndexOf($choices, $readers[$index].name)")
		if _, ok := raw.(int); !ok {
			t.Fatalf("index %d unresolved: %#v (%T)", i, raw, raw)
		}
		value, _, err := nativeCriterion(t, state, "$Len($choices) = 0 || $IndexOf($choices, $readers[$index].name) >= 0")
		if err != nil {
			t.Fatal(err)
		}
		if value == true {
			selected = append(selected, readers[i]["name"].(string))
		}
	}
	if len(selected) != 2 || selected[0] != "reader03" || selected[1] != "reader47" {
		t.Fatalf("selected %d readers: %#v", len(selected), selected)
	}
}
