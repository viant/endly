package webdriver

import (
	"fmt"
	smatcher "github.com/viant/endly/service/testing/runner/webdriver/matcher"
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
	"strings"
)

const (
	undefined int = iota
	eof
	illegal
	whitespaces
	id
	assign
	selector
	selectorBy
	params
	dot
	method
)

var whitespaceMatcher = parsly.NewToken(whitespaces, " ", matcher.NewWhiteSpace())
var idMatcher = parsly.NewToken(id, "IDENT", smatcher.NewIdentity())
var assignMatcher = parsly.NewToken(assign, "=", matcher.NewByte('='))
var dotMatcher = parsly.NewToken(dot, ".", matcher.NewByte('.'))
var selectorMatcher = parsly.NewToken(selector, "(...)", matcher.NewBlock('(', ')', '\\'))
var methodMatcher = parsly.NewToken(method, "Method", smatcher.NewLiteral())

// parser represents selenium command action parser
type parser struct{}

// Parse parses supplied expression. It returns criteria or parsing error.
func (p *parser) Parse(command string) (*Action, error) {
	result := &Action{
		Calls: []*MethodCall{{}},
	}
	cursor := parsly.NewCursor("", []byte(command), 0)
	var call = result.Calls[0]
	var webSelector WebSelector
	var callParams = ""

	expectTokens := []*parsly.Token{selectorMatcher, methodMatcher, idMatcher}

outer:
	for {

		match := cursor.MatchAfterOptional(whitespaceMatcher, expectTokens...)

		switch match.Token.Code {

		case selector:
			matched := match.Text(cursor)
			webSelector = WebSelector(matched[1 : len(matched)-1])
			match = cursor.MatchAfterOptional(whitespaceMatcher, dotMatcher)
			if match.Token.Code != dot {
				return nil, cursor.NewError(dotMatcher)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, methodMatcher, idMatcher)
			switch match.Token.Code {
			case method:
				matched = match.Text(cursor)
				index := strings.Index(matched, "(")
				call.Method = matched[:index]
				callParams = strings.Trim(matched[index+1:len(matched)-1], `'`)
				if len(callParams) > 0 {
					call.Parameters = []interface{}{callParams}
				}
			case id:
				call.Method = match.Text(cursor)
			default:
				return nil, cursor.NewError(methodMatcher, idMatcher)
			}
		case method:
			matched := match.Text(cursor)
			index := strings.Index(matched, "(")
			call.Method = matched[:index]
			callParams = strings.Trim(matched[index+1:len(matched)-1], `'`)
			if len(callParams) > 0 {
				call.Parameters = []interface{}{callParams}
			}

		case id:
			matched := match.Text(cursor)
			match = cursor.MatchAfterOptional(whitespaceMatcher, assignMatcher)
			if match.Token.Code == assign {
				result.Key = matched
			} else {
				call.Method = matched
			}
		case parsly.EOF:
			break outer
		default:
			return nil, cursor.NewError(expectTokens...)
		}
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

// NewParser creates a new criteria parser
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
