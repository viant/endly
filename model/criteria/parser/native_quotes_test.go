package parser

import (
	"github.com/viant/endly/model/criteria/ast"
	"testing"
)

func TestNativeQuotedStringLiterals(t *testing.T) {
	cases := []struct{ name, expression, value, quote string }{
		{"double", `"not_found" == "not_found"`, "not_found", `"`},
		{"single", `'not_found' == 'not_found'`, "not_found", `'`},
		{"double escaped quote", `"say \"hello\"" == "say \"hello\""`, `say \"hello\"`, `"`},
		{"single escaped quote", `'it\'s here' == 'it\'s here'`, `it\'s here`, `'`},
		{"double literal dollar", `"$ForeignDSL(keep)" == "$ForeignDSL(keep)"`, "$ForeignDSL(keep)", `"`},
		{"single literal dollar", `'$ForeignDSL(keep)' == '$ForeignDSL(keep)'`, "$ForeignDSL(keep)", `'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseCriteria(tc.expression)
			if err != nil {
				t.Fatal(err)
			}
			binary, ok := parsed.X.(*ast.Binary)
			if !ok {
				t.Fatalf("expected binary expression, got %T", parsed.X)
			}
			for _, node := range []interface{}{binary.X, binary.Y} {
				literal, ok := node.(*ast.Literal)
				if !ok || literal.Value != tc.value || literal.Quote != tc.quote {
					t.Fatalf("literal changed: %#v (%T), want value %q quote %q", node, node, tc.value, tc.quote)
				}
			}
		})
	}
}
