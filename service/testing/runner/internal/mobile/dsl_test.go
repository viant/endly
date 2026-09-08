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
