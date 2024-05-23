package parser

import (
	smatcher "github.com/viant/endly/model/criteria/matcher"
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
	"github.com/viant/parsly/matcher/option"
)

const (
	whitespaceCode int = iota
	parenthesesCode

	logicalOperator
	unaryOperator
	binaryOperator

	singleQuotedStringLiteral
	doubleQuotedStringLiteral
	boolLiteral

	stringLiteral

	numericLiteral
	selectorCode

	questionMark

	terminatorCode

	colon
)

var whitespaceMatcher = parsly.NewToken(whitespaceCode, "whitespace", matcher.NewWhiteSpace())
var parenthesesMatcher = parsly.NewToken(parenthesesCode, "()", matcher.NewBlock('(', ')', '\\'))

var unaryOperatorMatcher = parsly.NewToken(unaryOperator, "unary OPERATOR", matcher.NewSpacedSet([]string{"!", "not", "defined"}, &option.Case{}))
var binaryOperatorMatcher = parsly.NewToken(binaryOperator, "binary OPERATOR", matcher.NewSpacedSet([]string{"!=", ":!/", ":/", ":!", ":", ">=", "<=", "==", "=", ">", "<", "contains", "contains!"}, &option.Case{}))
var logicalOperatorMatcher = parsly.NewToken(logicalOperator, "AND|OR", matcher.NewSet([]string{"&&", "||"}, &option.Case{}))
var boolLiteralMatcher = parsly.NewToken(boolLiteral, "true|false", matcher.NewSet([]string{"true", "false"}, &option.Case{}))
var singleQuotedStringLiteralMatcher = parsly.NewToken(singleQuotedStringLiteral, `'...'`, matcher.NewByteQuote('\'', '\\'))
var doubleQuotedStringLiteralMatcher = parsly.NewToken(doubleQuotedStringLiteral, `"..."`, matcher.NewByteQuote('\'', '\\'))
var numericLiteralMatcher = parsly.NewToken(numericLiteral, `NUMERIC`, matcher.NewNumber())
var stringLiteralMatcher = parsly.NewToken(stringLiteral, `STRING`, smatcher.NewFragment())
var selectorMatcher = parsly.NewToken(selectorCode, "SELECTOR", smatcher.NewSelector())
var terminatorMatcher = parsly.NewToken(terminatorCode, "/", smatcher.NewTerminator('/', false))
var terminatorMatcherInc = parsly.NewToken(terminatorCode, "/", smatcher.NewTerminator('/', true))

var questionMarkMatcher = parsly.NewToken(questionMark, "?", matcher.NewByte('?'))
var colonMatcher = parsly.NewToken(colon, ":", matcher.NewByte(':'))
