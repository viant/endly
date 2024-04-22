package matcher

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
)

type literal struct {
}

// Match matches a string
func (n *literal) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	initPos := cursor.Pos
	pos := cursor.Pos
	if startsWithCharacter := matcher.IsLetter(input[pos]); startsWithCharacter {
		pos++
		matched++
	} else {
		return 0
	}

	size := len(input)
	for i := pos; i < size; i++ {
		switch input[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_':
			matched++
			continue
		case '(':
			block := matcher.NewBlock('(', ')', '\\')
			cursor.Pos = cursor.Pos + matched
			count := block.Match(cursor)
			cursor.Pos = initPos
			if count == 0 {
				return 0
			}
			return matched + count
		default:
			if matcher.IsLetter(input[i]) {
				matched++
				continue
			}
			return 0
		}
	}
	return 0
}

// Newliteral creates a string matcher
func NewLiteral() *literal {
	return &literal{}
}
