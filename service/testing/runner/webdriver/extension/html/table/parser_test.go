package table

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		name     string
		html     string
		expected [][]string
	}{
		{
			name: "Simple table",
			html: `
				<table>
					<tr><td>Name</td><td>Age</td></tr>
					<tr><td>Alice</td><td>30</td></tr>
					<tr><td>Bob</td><td>25</td></tr>
				</table>
			`,
			expected: [][]string{
				{"Name", "Age"},
				{"Alice", "30"},
				{"Bob", "25"},
			},
		},
		{
			name: "Empty table",
			html: `
				<table>
				</table>
			`,
			expected: nil,
		},
		{
			name: "Table with nested elements",
			html: `
				<table>
					<tr><td>Data <b>1</b></td><td>Data <i>2</i></td></tr>
				</table>
			`,
			expected: [][]string{
				{"Data 1", "Data 2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewParser(tc.html)
			if !assert.Nil(t, err) {
				return
			}
			result := parser.ParseTable()
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result: %v", err)
			}
			expectedJSON, err := json.Marshal(tc.expected)
			if err != nil {
				t.Fatalf("Failed to marshal expected: %v", err)
			}
			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("Test failed: expected %v, got %v", string(expectedJSON), string(resultJSON))
			}
		})
	}
}
