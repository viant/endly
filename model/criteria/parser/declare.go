package parser

import (
	"github.com/viant/endly/model/criteria/ast"
	"github.com/viant/parsly"
	"strings"
)

func parseDeclare(input string, err error, cursor *parsly.Cursor, when string, expr string, elseExpr string) (string, string, string, error) {
	binary := ast.Binary{}
	err = parseQualify(cursor, &binary, true, "")
	if err != nil {
		err = nil
		return "", "", input, nil
	}
	when = input[:cursor.Pos]

	match := cursor.MatchAfterOptional(whitespaceMatcher, questionMarkMatcher)
	if match.Code != questionMark {
		return "", input, "", nil
	}
	exprNode, err := expectOperand(cursor)
	if err != nil {
		return "", input, "", err
	}
	expr = exprNode.Stringify()
	match = cursor.MatchAfterOptional(whitespaceMatcher, colonMatcher)
	if match.Code == colon {
		elseNode, err := expectOperand(cursor)
		if err != nil {
			return "", input, "", err
		}
		elseExpr = elseNode.Stringify()
	}
	if when == "<" {
		return "", "", "", nil
	}
	if index := strings.Index(expr, ":"); index != -1 && elseExpr == "" {
		elseExpr = expr[index+1:]
		expr = expr[:index]
	}
	return when, expr, elseExpr, nil
}
