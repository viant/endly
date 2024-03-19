package adapter

import "strings"

var basicTypes = map[string]bool{
	"string":    true,
	"int":       true,
	"float64":   true,
	"float43":   true,
	"int64":     true,
	"int32":     true,
	"time.Time": true,
}

func isBasicType(candidate string) bool {
	candidate = strings.TrimSpace(strings.Replace(candidate, "*", "", len(candidate)))
	return basicTypes[candidate]
}
