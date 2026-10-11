package validator_test

import "testing"

func TestValidatorServiceSortTextActualValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		ids   []int
		valid bool
	}{
		{"reordered duplicates", []int{2, 1, 1}, true},
		{"changed value", []int{1, 1, 3}, false},
		{"changed duplicate count", []int{1, 2, 2}, false},
		{"missing value", []int{1, 2}, false},
		{"extra duplicate", []int{1, 1, 1, 2}, false},
		{"empty actual", []int{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			expected := map[string]interface{}{
				"@length@ids": 3,
				"ids":         []interface{}{map[string]interface{}{"@sortText@": true}, 1, 1, 2},
			}
			passed, err := assertWithService(expected, map[string]interface{}{"ids": test.ids})
			if test.valid && (err != nil || passed == 0) {
				t.Fatalf("reordered values must pass: passed=%d err=%v", passed, err)
			}
			if !test.valid && err == nil {
				t.Fatal("changed values or multiplicity must fail")
			}
		})
	}
}
