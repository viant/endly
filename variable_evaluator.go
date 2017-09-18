package endly

import (
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"unicode"
)

const (
	expectVariableStart = iota
	expectVariableName
	expectVariableNameEnclosureEnd
)

func Expand(state common.Map, text string) string {
	if strings.Index(text, "$") == -1 {
		return text
	}
	var expandVariable = func(variableName, result string) string {
		value, has := state.GetValue(string(variableName[1:]))
		if has {
			return result + toolbox.AsString(value)
		}
		return result + variableName
	}
	var variableName = ""
	var parsingState = expectVariableStart
	var result = ""

	for i, rune := range text {
		aChar := string(text[i : i+1])
		switch parsingState {
		case expectVariableStart:
			if aChar == "$" {
				variableName += aChar
				if i+1 < len(text) {
					nextChar := string(text[i+1 : i+2])
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
