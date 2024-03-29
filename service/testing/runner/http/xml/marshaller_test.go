package xml

import (
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed testdata/complex.xml
var complex []byte

//go:embed testdata/complex_expect.json
var expectedComplex []byte

func TestUnmarshalXML(t *testing.T) {
	testCases := []struct {
		desc       string // description of the test case
		xmldata    string
		expectJSON string
		expected   *Node
		wantErr    bool
	}{
		{
			desc:       "Test with complext XML",
			xmldata:    string(complex),
			expectJSON: string(expectedComplex),
		},
		{
			desc: "Test with attributes",
			xmldata: `<root>
				<item1 attr1="value1">This is item 1</item1>
				<item2>This is item 2</item2>
			</root>`,
			expected: &Node{
				Name:  "root",
				Attrs: map[string]string{},
				Children: []*Node{
					{Name: "item1", Attrs: map[string]string{"attr1": "value1"}, Value: "This is item 1"},
					{Name: "item2", Value: "This is item 2", Attrs: map[string]string{}}}},
			wantErr: false,
		},

		{
			desc:    "Test with malformed XML",
			xmldata: `<root><item1>This is item 1</item1`,
			wantErr: true,
		},
		// Add more test cases here as needed
	}

	for _, tc := range testCases[:1] {
		t.Run(tc.desc, func(t *testing.T) {
			var m Node
			if tc.expected == nil && len(tc.expectJSON) > 0 {
				err := json.Unmarshal([]byte(tc.expectJSON), &tc.expected)
				if err != nil {
					t.Errorf("UnmarshalXML() error = %v, wantErr %v", err, tc.wantErr)
					return
				}
			}

			err := xml.Unmarshal([]byte(tc.xmldata), &m)
			if (err != nil) != tc.wantErr {
				t.Errorf("UnmarshalXML() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && !assert.EqualValues(t, tc.expected, &m) {
				data, _ := json.Marshal(m)
				fmt.Printf("%s\n", data)
			}
		})
	}
}
