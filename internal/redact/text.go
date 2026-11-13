// Package redact contains redaction shared by request and manager diagnostics.
package redact

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var bearerPattern = regexp.MustCompile(`(?i)Bearer[\t ]+[A-Za-z0-9._~+/=-]+`)
var jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

// Text removes recognizable credentials from diagnostic text.
func Text(text string) string {
	return jwtPattern.ReplaceAllString(bearerPattern.ReplaceAllString(text, "Bearer [REDACTED]"), "[REDACTED]")
}

// RequestError removes request strings from converter diagnostics. Converter
// errors embed whole source containers and partially populated targets, so a
// value under an innocuous key (for example WebDriver Parameters) is sensitive
// too. Keep the conversion reason and types, without retaining its unsafe error
// as an unwrap target. Never invoke methods on arbitrary request values.
func RequestError(message string, request interface{}) string {
	values := map[string]bool{}
	remaining := 10000
	if !collectStrings(reflect.ValueOf(request), values, map[visit]bool{}, 0, &remaining) {
		return "request conversion failed (diagnostic traversal limit reached)"
	}
	ordered := make([]string, 0, len(values))
	for value := range values {
		ordered = append(ordered, value)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	return Text(replaceValues(message, ordered))
}

// Match short values as complete tokens rather than letters inside diagnostic
// words or type names. Longer values are removed even from prefixed messages.
func replaceValues(message string, values []string) string {
	var result strings.Builder
	for len(message) > 0 {
		start, length := len(message), 0
		for _, value := range values {
			for offset := 0; offset < len(message); {
				index := strings.Index(message[offset:], value)
				if index < 0 {
					break
				}
				index += offset
				end := index + len(value)
				if len(value) > 3 || (valueBoundary(message[:index], true) && valueBoundary(message[end:], false)) {
					if index < start || (index == start && len(value) > length) {
						start, length = index, len(value)
					}
					break
				}
				offset = index + 1
			}
		}
		if length == 0 {
			result.WriteString(message)
			break
		}
		result.WriteString(message[:start])
		result.WriteString("[REDACTED]")
		message = message[start+length:]
	}
	return result.String()
}

func valueBoundary(text string, before bool) bool {
	if text == "" {
		return true
	}
	var r rune
	if before {
		r, _ = utf8.DecodeLastRuneInString(text)
	} else {
		r, _ = utf8.DecodeRuneInString(text)
	}
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
}

type visit struct {
	kind    reflect.Kind
	typeOf  reflect.Type
	pointer uintptr
	length  int
}

func collectStrings(value reflect.Value, values map[string]bool, visited map[visit]bool, depth int, remaining *int) bool {
	if depth > 64 || *remaining <= 0 {
		return false
	}
	*remaining--
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Interface:
		if !value.IsNil() {
			return collectStrings(value.Elem(), values, visited, depth+1, remaining)
		}
	case reflect.Ptr, reflect.Map, reflect.Slice:
		if value.IsNil() {
			return true
		}
		var pointer uintptr
		if value.Kind() == reflect.Map {
			pointer = uintptr(value.UnsafePointer())
		} else {
			pointer = value.Pointer()
		}
		identity := visit{kind: value.Kind(), typeOf: value.Type(), pointer: pointer}
		if value.Kind() == reflect.Slice {
			identity.length = value.Len()
		}
		if visited[identity] {
			return true
		}
		visited[identity] = true
		switch value.Kind() {
		case reflect.Ptr:
			return collectStrings(value.Elem(), values, visited, depth+1, remaining)
		case reflect.Map:
			iter := value.MapRange()
			for iter.Next() {
				if !collectStrings(iter.Key(), values, visited, depth+1, remaining) || !collectStrings(iter.Value(), values, visited, depth+1, remaining) {
					return false
				}
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				if !collectStrings(value.Index(i), values, visited, depth+1, remaining) {
					return false
				}
			}
		}
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !collectStrings(value.Index(i), values, visited, depth+1, remaining) {
				return false
			}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !collectStrings(value.Field(i), values, visited, depth+1, remaining) {
				return false
			}
		}
	case reflect.String:
		text := value.String()
		if text == "" {
			return true
		}
		values[text] = true
		quoted := strconv.Quote(text)
		values[quoted[1:len(quoted)-1]] = true
		quoted = strconv.QuoteToASCII(text)
		values[quoted[1:len(quoted)-1]] = true
		encoded, _ := json.Marshal(text)
		values[string(encoded[1:len(encoded)-1])] = true
	}
	return true
}
