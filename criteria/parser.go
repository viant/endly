package criteria

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

const (
	undefined int = iota
	eof
	illegal
	whitespaces
	operand
	operator
	logicalOperator
	assertlyExprMatcher
	quoted
	jsonObject
	jsonArray
	grouping
)

var matchers = map[int]toolbox.Matcher{
	eof:         toolbox.EOFMatcher{},
	whitespaces: toolbox.CharactersMatcher{" \n\t"},
	operand:     toolbox.NewCustomIdMatcher(".", "_", "$", "[", "]", "{", "}", "!", "-", "/", "\\", "+", "-", "*"),
	operator: toolbox.KeywordsMatcher{
		Keywords:      []string{"=", ">=", "<=", "<>", ">", "<", "!=", ":"},
		CaseSensitive: false,
	},
	logicalOperator: toolbox.KeywordsMatcher{
		Keywords:      []string{"&&", "||"},
		CaseSensitive: false,
	},
	quoted:              &toolbox.BodyMatcher{"'", "'"},
	grouping:            &toolbox.BodyMatcher{"(", ")"},
	jsonObject:          &toolbox.BodyMatcher{"{", "}"},
	jsonArray:           &toolbox.BodyMatcher{"[", "]"},
	assertlyExprMatcher: toolbox.NewSequenceMatcher("&&", "||"),
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
func (p *Parser) Parse(expression string) (*Predicate, error) {
	result := NewPredicate("")
	tokenizer := toolbox.NewTokenizer(expression, illegal, eof, matchers)
	var criterion *Criterion

	var leftOperandTokens = []int{quoted, jsonObject, jsonArray, grouping, operand, operator}
	var rightOperandTokens = []int{quoted, jsonObject, jsonArray, operand, operator}

	parsingCriteria := result

	setUniOperandCriteriaIfNeeed := func(criterion *Criterion) {
		if criterion.Operator != "" || criterion.RightOperand != nil {
			return
		}
		criterion.Operator = "!="
	}

outer:
	for {

		expectedTokens := leftOperandTokens
		if criterion != nil && criterion.Operator != "" {
			expectedTokens = rightOperandTokens
		}
		token, err := p.expectOptionalWhitespaceFollowedBy(tokenizer, "id or grouping expression", expectedTokens...)
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
					Predicate: criteria,
				})
			}

		case operand, jsonObject, jsonArray, quoted:
			var matched = token.Matched
			if token.Token == quoted {
				matched = strings.Trim(token.Matched, "' ")
			}
			token = tokenizer.Next(grouping)
			if token != nil {
				matched += token.Matched
			}
			criterion = &Criterion{
				LeftOperand: matched,
			}
			parsingCriteria.Criteria = append(parsingCriteria.Criteria, criterion)
		case operator:
			criterion = &Criterion{}
			parsingCriteria.Criteria = append(parsingCriteria.Criteria, criterion)
		}

		if token.Token != operator {
			token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "operator", operator, logicalOperator, eof)
			if err != nil {
				return nil, err
			}
		}

		if token.Token == eof {
			break outer
		} else if token.Token == operator {

			leftOperand := toolbox.AsString(criterion.LeftOperand)
			if strings.HasSuffix(leftOperand, "!") {
				criterion.LeftOperand = string(leftOperand[:len(leftOperand)-1])
				criterion.Operator = "!" + token.Matched
			} else {
				criterion.Operator = token.Matched
			}

			if criterion.Operator == ":" {
				token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "right operand", assertlyExprMatcher, eof)
			} else {
				token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "right operand", quoted, jsonObject, jsonArray, operand, eof)
			}
			if err != nil {
				return nil, err
			}
			if token.Token == eof {
				break outer
			}
			var matched = token.Matched
			if token.Token == quoted {
				matched = strings.Trim(token.Matched, "' ")
			}
			criterion.RightOperand = matched
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
		parsingCriteria = NewPredicate(token.Matched)
		conjunctionCriterion.Predicate = parsingCriteria
		setUniOperandCriteriaIfNeeed(criterion)
		criterion = nil
	}
	setUniOperandCriteriaIfNeeed(criterion)
	return result, nil
}

//NewParser creates a new criteria parser
func NewParser() *Parser {
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
