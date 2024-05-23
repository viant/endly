package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDeclare(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedWhen  string
		expectedExpr  string
		expectedElse  string
		expectedError string
	}{
		{
			input:        "${httpTrips.Response[0].Body}://auctionwon/",
			expectedWhen: "",
			expectedExpr: "${httpTrips.Response[0].Body}://auctionwon/",
			expectedElse: "",
		},

		{
			input:        "$params.profile:/$/?dev:${params.profile}", //
			expectedWhen: "$params.profile:/$/",
			expectedExpr: "dev",
			expectedElse: "${params.profile}",
		},

		{
			name:         "valid declaration with else",
			input:        "x > 5 ? y : z",
			expectedWhen: "x > 5 ",
			expectedExpr: "y",
			expectedElse: "z",
		},
		{
			name:         "valid declaration without else",
			input:        "a == b ? c",
			expectedWhen: "a == b ",
			expectedExpr: "c",
			expectedElse: "",
		},

		{
			name:         "basic declaration",
			input:        "$c",
			expectedWhen: "",
			expectedExpr: "$c",
			expectedElse: "",
		},

		{
			name:         "invalid declaration {}",
			input:        "{}",
			expectedWhen: "",
			expectedExpr: "{}",
			expectedElse: "",
		},
		{
			name:         "invalid declaration <>",
			input:        "<?:>",
			expectedWhen: "",
			expectedExpr: "<?:>",
			expectedElse: "",
		},

		// More test cases, especially edge cases and invalid inputs
	}

	for _, tc := range tests[:1] {
		t.Run(tc.name, func(t *testing.T) {
			when, expr, elseExpr, err := ParseDeclaration(tc.input)
			if tc.expectedError != "" {
				assert.ErrorContains(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedWhen, when)
				assert.Equal(t, tc.expectedExpr, expr)
				assert.Equal(t, tc.expectedElse, elseExpr)
			}
		})
	}
}
