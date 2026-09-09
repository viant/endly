package mobile

import (
	"reflect"
	"testing"
)

func TestParseDSL(t *testing.T) {
	tests := []struct {
		source string
		want   *DSLCommand
	}{
		{
			source: `app.getByTestId("email").fill("a,b@example.test")`,
			want: &DSLCommand{Namespace: "app", Calls: []Call{
				{Name: "getByTestId", Args: []interface{}{"email"}},
				{Name: "fill", Args: []interface{}{"a,b@example.test"}},
			}},
		},
		{
			source: `greeting = app.getByText('Welcome').text()`,
			want: &DSLCommand{Key: "greeting", Namespace: "app", Calls: []Call{
				{Name: "getByText", Args: []interface{}{"Welcome"}},
				{Name: "text"},
			}},
		},
		{
			source: `expect(app.getByTestId("done")).toBeVisible(15000)`,
			want: &DSLCommand{Namespace: "app", Expectation: true, Calls: []Call{
				{Name: "getByTestId", Args: []interface{}{"done"}},
				{Name: "toBeVisible", Args: []interface{}{int64(15000)}},
			}},
		},
		{
			source: `device.setLocation(34.1, -118.2, true)`,
			want: &DSLCommand{Namespace: "device", Calls: []Call{
				{Name: "setLocation", Args: []interface{}{34.1, -118.2, true}},
			}},
		},
	}
	for _, test := range tests {
		got, err := ParseDSL(test.source)
		if err != nil {
			t.Fatalf("parse %q: %v", test.source, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("parse %q\n got: %#v\nwant: %#v", test.source, got, test.want)
		}
	}
}

func TestParseDSLRejectsAmbiguousSyntax(t *testing.T) {
	for _, source := range []string{"", "page.goto()", "a-b = app.getByText('x').tap()", "expect(app.getByText('x'))"} {
		if _, err := ParseDSL(source); err == nil {
			t.Fatalf("expected %q to fail", source)
		}
	}
}

func TestParseTypedInstructions(t *testing.T) {
	tests := []struct {
		candidate interface{}
		want      *DSLCommand
	}{
		{
			candidate: map[string]interface{}{
				"key":     "second",
				"locator": map[string]interface{}{"strategy": "accessibilityId", "value": "row", "selection": "nth", "index": 1},
				"action":  map[string]interface{}{"name": "text"},
			},
			want: &DSLCommand{Key: "second", Namespace: "app", Calls: []Call{
				{Name: "locator", Args: []interface{}{"accessibilityId", "row"}},
				{Name: "nth", Args: []interface{}{int64(1)}},
				{Name: "text"},
			}},
		},
		{
			candidate: TypedCommand{Expect: &TypedExpectation{
				Locator: &TypedLocator{Strategy: "resourceId", Value: "status"},
				Matcher: TypedCall{Name: "toHaveText", Args: []interface{}{"Ready"}}, TimeoutMs: 500,
			}},
			want: &DSLCommand{Namespace: "app", Expectation: true, Calls: []Call{
				{Name: "locator", Args: []interface{}{"resourceId", "status"}},
				{Name: "toHaveText", Args: []interface{}{"Ready", int64(500)}},
			}},
		},
		{
			candidate: TypedCommand{Key: "contexts", Device: &TypedCall{Name: "contexts"}},
			want:      &DSLCommand{Key: "contexts", Namespace: "device", Calls: []Call{{Name: "contexts"}}},
		},
	}
	for _, test := range tests {
		got, _, err := ParseInstruction(test.candidate)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("typed command mismatch\n got: %#v\nwant: %#v", got, test.want)
		}
	}
}

func TestTypedInstructionRejectsAmbiguousShape(t *testing.T) {
	_, _, err := ParseInstruction(TypedCommand{
		Locator: &TypedLocator{Strategy: "id", Value: "x"}, Action: &TypedCall{Name: "tap"},
		Device: &TypedCall{Name: "home"},
	})
	if err == nil {
		t.Fatal("expected ambiguous typed command to fail")
	}
}
