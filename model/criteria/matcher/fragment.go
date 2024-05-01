package matcher

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
)

type fragment struct{}

// Match matches a string
func (n *fragment) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	if startsWithCharacter := matcher.IsLetter(input[pos]); startsWithCharacter {
		pos++
		matched++
	} else {
		return 0
	}
	arrayDepth := 0
	objectDepth := 0
	size := len(input)
	for i := pos; i < size; i++ {
		matched++
		if objectDepth == 0 && arrayDepth == 0 {
			if matcher.IsWhiteSpace(input[i]) {
				matched--
				return matched
			}
		}

		switch input[i] {
		case '[':
			if objectDepth == 0 {
				arrayDepth++
			}
		case ']':

			if objectDepth == 0 {
				arrayDepth--
				if arrayDepth == 0 {
					break
				}
			}
		case '{':
			if arrayDepth == 0 {
				objectDepth++
			}
		case '}':
			if arrayDepth == 0 {
				objectDepth--
				if objectDepth == 0 {
					break
				}
			}
		default:
		}
	}
	return matched
}

// NewFragment creates a string matcher
func NewFragment() *fragment {
	return &fragment{}
}
