package endly

import (
	"unicode"
	"strings"
	"github.com/viant/endly/common"
)

const (
	expectVariableStart = iota
	expectVariableName

	expectVariableNameEnclosureEnd
)

func extractState(state common.Map, name string) (string, bool) {

	name = string(name[1:])

	if name == "" {
		return "", false
	}

	if string(name[0:1]) == "{" {
		name = name[1:len(name)-1]
	}
	if strings.Contains(name, ".") {
		fragments := strings.Split(name, ".")
		for i, fragment := range fragments {
			isLast := i+1 == len(fragments)
			if isLast {
				name = fragment
			} else {
				state = state.GetMap(fragment)
				if state == nil {
					return "", false
				}
			}

		}
	}
	if state.Has(name) {
		return state.GetString(name), true
	}
	return "", false
}

func Expand(state common.Map, text string) string {
	if strings.Index(text, "$") == -1 {
		return text
	}

	var expandVariable = func(variableName, result string) string {
		value, has := extractState(state, variableName)
		if has {
			return result + value
		}
		return result + variableName
	}

	var variableName = "";
	var parsingState = expectVariableStart
	var result = ""

	for i, rune := range text {

		aChar := string(text[i:i+1])

		switch parsingState {
		case expectVariableStart:
			if aChar == "$" {
				variableName += aChar
				if i+1 < len(text) {
					nextChar := string(text[i+1:i+2])
					if nextChar == "{" {
						parsingState = expectVariableNameEnclosureEnd
						continue

					}
				}
				parsingState = expectVariableName
				continue
			}
			result += aChar

		case expectVariableNameEnclosureEnd:
			variableName += aChar
			if aChar != "}" {
				continue
			}
			result = expandVariable(variableName, result)
			variableName = ""
			parsingState = expectVariableStart

		case expectVariableName:
			if unicode.IsLetter(rune) || unicode.IsDigit(rune) || aChar == "." {
				variableName += aChar
				continue
			}
			result = expandVariable(variableName, result)
			result += aChar
			variableName = ""
			parsingState = expectVariableStart

		}
	}
	if len(variableName) > 0 {
		result = expandVariable(variableName, result)
	}
	return result
}
