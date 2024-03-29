package log

import (
	"fmt"
	"regexp"
)

func matchLogIndex(expr *regexp.Regexp, input string) string {
	if expr.MatchString(input) {
		matches := expr.FindStringSubmatch(input)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func logTypeMetaKey(name string) string {
	return fmt.Sprintf("meta_%v", name)
}
