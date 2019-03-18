package shared

import (
	"fmt"
	"strings"
)

//ToSelector returns selector
func ToSelector(selectorMap map[string]string) string {
	var selector = make([]string, 0)
	for k, v := range selectorMap {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	return strings.Join(selector, ",")
}
