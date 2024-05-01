package parser

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/model/criteria/ast"
	"testing"
)

func TestParseCriteria(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ast.Qualify
		err      string
	}{

		{
			name:  "simple equality",
			input: `a == b`,
			expected: &ast.Qualify{
				X: &ast.Binary{
					X:  &ast.Literal{Value: "a", Type: "string"},
					Op: "==",
					Y:  &ast.Literal{Value: "b", Type: "string"},
				},
			},
		},
		{
			name:  "simple equation",
			input: `$a == b`,
			expected: &ast.Qualify{
				X: &ast.Binary{
					X:  &ast.Selector{X: "$a"},
					Op: "==",
					Y:  &ast.Literal{Value: "b", Type: "string"},
				},
			},
		},

		{
			name:  "simple unary",
			input: `$a`,
			expected: &ast.Qualify{
				X: &ast.Unary{
					X: &ast.Selector{X: "$a"},
				},
			},
		},

		{
			name:  "unary negation",
			input: `!$a`,
			expected: &ast.Qualify{
				X: &ast.Unary{X: &ast.Selector{X: "$a"}, Op: "!"},
			},
		},

		{
			name:  "defined unary",
			input: `defined $a`,
			expected: &ast.Qualify{
				X: &ast.Unary{X: &ast.Selector{X: "$a"}, Op: "defined"},
			},
		},
		// Add more test cases here for different expressions and expected outcomes
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseCriteria(tc.input)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}
