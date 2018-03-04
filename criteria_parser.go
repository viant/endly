package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

const (
	undefined int = iota
	eof
	illegal
	whitespaces
	id
	operator
	logicalOperator
	assertlyExprMatcher
	jsonObject
	jsonArray
	grouping
)

var matchers = map[int]toolbox.Matcher{
	eof:         toolbox.EOFMatcher{},
	whitespaces: toolbox.CharactersMatcher{" \n\t"},
	id:          toolbox.NewCustomIdMatcher(".", "_", "$", "[", "]", "{", "}", "!", "-"),
	operator: toolbox.KeywordsMatcher{
		Keywords:      []string{"=", ">=", "<=", "<>", ">", "<", "!=", ":"},
		CaseSensitive: false,
	},
	logicalOperator: toolbox.KeywordsMatcher{
		Keywords:      []string{"&&", "||"},
		CaseSensitive: false,
	},
	grouping:            toolbox.BodyMatcher{"(", ")"},
	jsonObject:          toolbox.BodyMatcher{"{", "}"},
	jsonArray:           toolbox.BodyMatcher{"[", "]"},
	assertlyExprMatcher: toolbox.NewSequenceMatcher("&&", "||", "("),
}

//Parser represents endly criteria parser
type Parser struct{}

func (p *Parser) expectOptionalWhitespaceFollowedBy(tokenizer *toolbox.Tokenizer, expectedTokensMessage string, expected ...int) (*toolbox.Token, error) {
	var expectedTokens = make([]int, 0)
	expectedTokens = append(expectedTokens, whitespaces)
	expectedTokens = append(expectedTokens, expected...)

	token := tokenizer.Nexts(expectedTokens...)

	if token.Token == eof && !toolbox.HasSliceAnyElements(expectedTokens, eof) {
		return nil, newIllegalTokenParsingError(tokenizer.Index, expectedTokensMessage)
	}

	if token.Token == illegal {
		return nil, newIllegalTokenParsingError(tokenizer.Index, expectedTokensMessage)
	}
	if token.Token == whitespaces {
		token = tokenizer.Nexts(expected...)
	}
	if token.Token == illegal {
		return nil, newIllegalTokenParsingError(tokenizer.Index, expectedTokensMessage)
	}
	if token.Token == eof && len(token.Matched) > 0 {
		return nil, newIllegalTokenParsingError(tokenizer.Index, expectedTokensMessage)
	}
	return token, nil
}

//Parse parses supplied expression. It returns criteria or parsing error.
func (p *Parser) Parse(expression string) (*Criteria, error) {
	result := NewCriteria("")
	tokenizer := toolbox.NewTokenizer(expression, illegal, eof, matchers)
	var criterion *Criterion

	parsingCriteria := result
outer:
	for {
		token, err := p.expectOptionalWhitespaceFollowedBy(tokenizer, "id or grouping expression", jsonObject, jsonArray, grouping, id)
		if err != nil {
			return nil, err
		}
		switch token.Token {
		case grouping:
			groupingExpression := string(token.Matched[1 : len(token.Matched)-1])
			criteria, err := p.Parse(groupingExpression)
			if err != nil {
				return nil, err
			}

			if len(parsingCriteria.Criteria) == 0 {
				parsingCriteria.Criteria = criteria.Criteria
				parsingCriteria.LogicalOperator = criteria.LogicalOperator
			} else {
				parsingCriteria.Criteria = append(parsingCriteria.Criteria, &Criterion{
					Criteria: criteria,
				})
			}
		case id, jsonObject:
			criterion = &Criterion{
				LeftOperand: token.Matched,
			}
			parsingCriteria.Criteria = append(parsingCriteria.Criteria, criterion)
		}
		token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "operator", operator, logicalOperator, eof)
		if err != nil {
			return nil, err
		}
		if token.Token == eof {
			break outer
		} else if token.Token == operator {

			criterion.Operator = token.Matched
			if criterion.Operator == ":" {
				token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "right operand", assertlyExprMatcher, eof)
			} else {
				token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "right operand", jsonObject, jsonArray, id, eof)
			}
			if err != nil {
				return nil, err
			}
			if token.Token == eof {
				break outer
			}

			criterion.RightOperand = token.Matched
			token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "logical conjunction", logicalOperator, eof)
			if err != nil {
				return nil, err
			}
			if token.Token == eof {
				break outer
			}
		}

		if parsingCriteria.LogicalOperator == "" {
			parsingCriteria.LogicalOperator = token.Matched
		}

		if parsingCriteria.LogicalOperator == token.Matched {
			continue
		}
		conjunctionCriterion := &Criterion{}
		parsingCriteria.Criteria = append(parsingCriteria.Criteria, conjunctionCriterion)
		parsingCriteria = NewCriteria(token.Matched)
		conjunctionCriterion.Criteria = parsingCriteria
		criterion = nil
	}
	return result, nil
}

//NewCriteriaParser creates a new criteria parser
func NewCriteriaParser() *Parser {
	return &Parser{}
}

type illegalTokenParsingError struct {
	Index    int
	Expected string
	error    string
}

func (e illegalTokenParsingError) Error() string {
	return e.error
}

func newIllegalTokenParsingError(index int, expected string) error {
	return &illegalTokenParsingError{Index: index, Expected: expected, error: fmt.Sprintf("illegal token at %v, expected %v", index, expected)}
}
