package parser

import (
	"github.com/viant/endly/model/criteria/ast"
	"github.com/viant/parsly"
)


func ParseDeclaration(input string) (when string, thenExpr string, elseExpr string, err error) {
	cursor := parsly.NewCursor("", []byte(input), 0)
	return parseDeclare(input, err, cursor, when, thenExpr, elseExpr)
}


func ParseCriteria(input string) (*ast.Qualify, error) {
	cursor := parsly.NewCursor("", []byte(input), 0)
	qualify := &ast.Qualify{}
	err := parseCriteria(cursor, qualify)
	if err != nil {
		return nil, err
	}
	return qualify, nil
}

