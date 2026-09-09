package mobile

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

var hierarchyAttributes = []string{
	"resource-id", "content-desc", "text", "name", "label", "value", "type", "class", "bounds", "enabled", "visible",
}

// SummarizeHierarchy converts Android UiAutomator or iOS WDA XML into a compact,
// grep-friendly tree. The filter is case-insensitive and maxNodes bounds output.
func SummarizeHierarchy(source, filter string, maxNodes int) (string, error) {
	if maxNodes <= 0 {
		maxNodes = 500
	}
	filter = strings.ToLower(strings.TrimSpace(filter))
	decoder := xml.NewDecoder(strings.NewReader(source))
	depth := 0
	matched := 0
	lines := []string{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode hierarchy XML: %w", err)
		}
		switch actual := token.(type) {
		case xml.StartElement:
			attributes := map[string]string{}
			for _, attribute := range actual.Attr {
				attributes[strings.ToLower(attribute.Name.Local)] = attribute.Value
			}
			parts := []string{actual.Name.Local}
			for _, name := range hierarchyAttributes {
				if value := strings.TrimSpace(attributes[name]); value != "" {
					parts = append(parts, name+"="+quoteSummary(value))
				}
			}
			line := strings.Repeat("  ", depth) + strings.Join(parts, " ")
			if filter == "" || strings.Contains(strings.ToLower(line), filter) {
				lines = append(lines, line)
				matched++
				if matched >= maxNodes {
					lines = append(lines, fmt.Sprintf("... truncated after %d matching nodes", maxNodes))
					return strings.Join(lines, "\n"), nil
				}
			}
			depth++
		case xml.EndElement:
			if depth > 0 {
				depth--
			}
		}
	}
	return strings.Join(lines, "\n"), nil
}

func quoteSummary(value string) string {
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func SortedDataLines(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s = %v", key, values[key]))
	}
	return result
}
