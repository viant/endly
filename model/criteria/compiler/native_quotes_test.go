package compiler_test

import (
	"github.com/viant/endly"
	"testing"
)

func TestNativeQuotedStringComparisons(t *testing.T) {
	state := endly.NewDefaultState(nil)
	state["status"] = "not_found"
	state["literal"] = "$ForeignDSL(keep)"
	state["escaped"] = `say \"hello\"`
	for _, tc := range []struct {
		expression string
		want       bool
	}{
		{`$status == "not_found"`, true},
		{`$status == 'not_found'`, true},
		{`$status != "not_found"`, false},
		{`$status == "found"`, false},
		{`"say \"hello\"" == "say \"hello\""`, true},
		{`$escaped == "say \"hello\""`, true},
		{`$literal == "$ForeignDSL(keep)"`, true},
		{`$literal == '$ForeignDSL(keep)'`, true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			actual, _, err := nativeCriterion(t, state, tc.expression)
			if err != nil || actual != tc.want {
				t.Fatalf("%s -> %#v (%T), err=%v; want %t", tc.expression, actual, actual, err, tc.want)
			}
		})
	}
	if state["literal"] != "$ForeignDSL(keep)" {
		t.Fatal("quoted dollar literal expanded or mutated")
	}
}
