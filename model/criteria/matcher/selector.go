package matcher

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
)

type selector struct{}

// Match matches a string
func (n *selector) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	if startsWithCharacter := input[pos] == '$'; startsWithCharacter {
		pos++
		matched++
	} else {
		return 0
	}
	depth := 0
	depthBracket := 0
	size := len(input)
	for i := pos; i < size; i++ {
		switch input[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_', '[', ']', '.':
			matched++
			continue
		case '(':
			if depth == 0 {
				depthBracket++
			}
			matched++
		case ')':
			matched++
			if depth == 0 {
				depthBracket--
				if depthBracket == 0 {
					return matched
				}
			}
		case '{':
			if depthBracket == 0 {
				depth++
			}
			matched++
		case '}':
			matched++
			if depthBracket == 0 {
				depth--
				if depth == 0 {
					return matched
				}
			}
		default:
			if depth > 0 || depthBracket > 0 {
				matched++
				continue
			}
			if matcher.IsLetter(input[i]) {
				matched++
				continue
			}

			return matched
		}
	}

	return matched
}

// NewSelector creates a string matcher
func NewSelector() *selector {
	return &selector{}
}
