package selenium

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
	id
	operator
	selector
	selectorBy
	params
	dot
	method
)

var matchers = map[int]toolbox.Matcher{
	eof:         toolbox.EOFMatcher{},
	whitespaces: toolbox.CharactersMatcher{" \n\t"},

	id: toolbox.NewCustomIdMatcher("_", "-"),
	operator: toolbox.KeywordsMatcher{
		Keywords:      []string{"="},
		CaseSensitive: false,
	},
	selector:   &toolbox.BodyMatcher{"(", ")"},
	selectorBy: toolbox.NewSequenceMatcher(":"),
	params:     &toolbox.BodyMatcher{"(", ")"},
	dot:        toolbox.CharactersMatcher{Chars: "."},
	method:     toolbox.LiteralMatcher{},
}

//parser represents selenium command action parser
type parser struct{}

func (p *parser) expectOptionalWhitespaceFollowedBy(tokenizer *toolbox.Tokenizer, expectedTokensMessage string, expected ...int) (*toolbox.Token, error) {
	var expectedTokens = make([]int, 0)
	expectedTokens = append(expectedTokens, whitespaces)
	expectedTokens = append(expectedTokens, expected...)

	token := tokenizer.Nexts(expectedTokens...)

	if token.Token == eof && !toolbox.HasSliceAnyElements(expectedTokens, eof) {
		return nil, newIllegalTokenParsingError(tokenizer.Index, token.Token, expectedTokensMessage)
	}

	if token.Token == illegal {
		return nil, newIllegalTokenParsingError(tokenizer.Index, token.Token, expectedTokensMessage)
	}
	if token.Token == whitespaces {
		token = tokenizer.Nexts(expected...)
	}
	if token.Token == illegal {
		return nil, newIllegalTokenParsingError(tokenizer.Index, token.Token, expectedTokensMessage)
	}
	if token.Token == eof && len(token.Matched) > 0 {
		return nil, newIllegalTokenParsingError(tokenizer.Index, token.Token, expectedTokensMessage)
	}
	return token, nil
}

//Parse parses supplied expression. It returns criteria or parsing error.
func (p *parser) Parse(command string) (*Action, error) {
	result := &Action{
		Calls: []*MethodCall{{}},
	}
	tokenizer := toolbox.NewTokenizer(command, illegal, eof, matchers)

	var call = result.Calls[0]
	var webSelector WebSelector
	var callParams = ""

	expectTokens := []int{selector, id}

outer:
	for {

		token, err := p.expectOptionalWhitespaceFollowedBy(tokenizer, "id/method or web element selector", expectTokens...)
		if err != nil {
			return nil, err
		}
		switch token.Token {

		case selector:
			webSelector = WebSelector(token.Matched[1 : len(token.Matched)-1])
			token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "'.' method selector", dot)
			if token.Token != dot {
				return nil, err
			}
			token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "'.' method", id)
			if token.Token != id {
				return nil, err
			}
			call.Method = token.Matched
			fallthrough
		case id:
			identity := token.Matched
			token, err = p.expectOptionalWhitespaceFollowedBy(tokenizer, "'operator/params", operator, params, eof)
			if err != nil {
				return nil, err
			}
			switch token.Token {
			case params:
				call.Method = identity
				if len(token.Matched) > 2 {
					callParams = string(token.Matched[1 : len(token.Matched)-1])
				}
				break outer
			case operator:
				if result.Key != "" {
					return nil, fmt.Errorf("invalid token operator, expected: params at %d", tokenizer.Index)
				}
				result.Key = identity
				continue
			case eof:
				call.Method = identity
				break outer
			default:

				return nil, fmt.Errorf("invalid token: '%v', %v", token.Token, err)
			}
		}
	}
	var params = strings.TrimSpace(callParams)
	if len(params) > 0 {
		call.Parameters = []interface{}{strings.Trim(callParams, " '\"")}
	}

	if len(webSelector) > 0 {
		result.Selector = &WebElementSelector{}
		result.Selector.By, result.Selector.Value = webSelector.ByAndValue()
		result.Selector.Key = result.Key
	}
	method := call.Method
	if len(method) > 1 {
		call.Method = strings.ToUpper(string(method[0])) + string(method[1:])
	}
	return result, nil
}

//NewParser creates a new criteria parser
func NewParser() *parser {
	return &parser{}
}

type illegalTokenParsingError struct {
	Index    int
	Expected string
	error    string
}

func (e illegalTokenParsingError) Error() string {
	return e.error
}

func newIllegalTokenParsingError(index, token int, expected string) error {
	return &illegalTokenParsingError{Index: index, Expected: expected, error: fmt.Sprintf("illegal token:%v at %v, expected %v", token, index, expected)}
}
