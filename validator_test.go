package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func TestValidator_Assert(t *testing.T) {

	type useCase struct {
		Expected interface{}
		Actual   interface{}
		Passable bool
	}

	useCases := []*useCase{
		{Expected: "abc", Actual: "abc", Passable: true},
		{Expected: "abc", Actual: "abcd", Passable: false},
		{Expected: "/abc/", Actual: "abcd", Passable: true},
		{Expected: "/!abc/", Actual: "abcd", Passable: false},
		{Expected: "~/.+(\\d+).+/", Actual: "avc1erwer", Passable: true},
		{Expected: "~/!.+(\\d+).+/", Actual: "avc1erwer", Passable: false},
		{Expected: "123.4343", Actual: 123.4343, Passable: true},
		{Expected: `{"a":1,"b":2}
{"z":1,"y":2}
`, Actual: `{"a":1,"b":2}
{"z":1,"y":2}
`, Passable: true},
		{Expected: `{"id":1,"b":2}
{"id":10,"y":2}
`, Actual: `{"id":10,"y":2}
{"id":1,"b":2}
`, Passable: false},
		{Expected: `{"@indexBy@":"k"}
{"k":1,"b":2}
{"k":10,"y":2}
`, Actual: `{"k":10,"y":2}
{"k":1,"b":2}
`, Passable: true},
	}

	validator := endly.Validator{}
	for _, test := range useCases {
		var info = &endly.ValidationInfo{}
		validator.Assert(test.Expected, test.Actual, info, "")
		if test.Passable {
			assert.True(t, info.TestPassed > 0 && info.TestFailed == 0, fmt.Sprintf("assert: %v %v", test.Expected, test.Actual))
		}

	}

}
