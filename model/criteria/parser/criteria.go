package parser

import (
	"github.com/viant/endly/model/criteria/ast"
	"github.com/viant/parsly"
)

// ParseCriteria parses qualify expr
func parseCriteria(cursor *parsly.Cursor, qualify *ast.Qualify) error {
	binary := &ast.Binary{}
	err := parseQualify(cursor, binary, false)
	if binary.Op == "" && binary.Y == nil && binary.X != nil {
		if unary, ok := binary.X.(*ast.Unary);ok {
			qualify.X = unary
		} else {
			qualify.X = &ast.Unary{X: binary.X, Op: ""}
		}
		return err
	}

	qualify.X = binary
	return err
}

func parseQualify(cursor *parsly.Cursor, binary *ast.Binary, withDeclare bool) (err error) {

	if withDeclare {
		pos := cursor.Pos
		match := cursor.MatchAfterOptional(whitespaceMatcher, questionMarkMatcher, colonMatcher)
		switch match.Code {
		case questionMark, colon:
			cursor.Pos = pos
			return
		}
	}

	if binary.X == nil {
		binary.X, err = expectOperand(cursor)
		if err != nil || binary == nil {
			return err
		}
	}
	if binary.Op == "" {
		match := cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, unaryOperatorMatcher, binaryOperatorMatcher, logicalOperatorMatcher)
		op := match.Text(cursor)
		switch match.Code {
		case unaryOperator:
			binary.Op = op
		case logicalOperator:
			binary.Op = op
		case binaryOperator:
			binary.Op = op

		default:
			return nil
		}
	}

	if binary.Y == nil {
		yExpr := &ast.Binary{}
		if err := parseQualify(cursor, yExpr, withDeclare); err != nil {
			return err
		}
		if yExpr.X != nil {
			binary.Y = yExpr
		}
		if yExpr.Op == "" && yExpr.Y == nil {
			binary.Y = yExpr.X
		}
	}
	normalizeBinary(binary)
	return nil
}

func normalizeBinary(binary *ast.Binary) {
	switch binary.Op {
	case "&&", "||":
	default:
		switch yOp := binary.Y.(type) {
			case *ast.Binary:
				swap := &ast.Binary{X: binary.X, Op: binary.Op, Y: yOp.X}
				binary.Op = yOp.Op
				binary.X = swap
				binary.Y = yOp.Y
				if binExpr, ok := binary.Y.(*ast.Binary); ok {
					normalizeBinary(binExpr)
				}

		}
	}
}

var operands = []*parsly.Token{
	boolLiteralMatcher,
	doubleQuotedStringLiteralMatcher,
	singleQuotedStringLiteralMatcher,
	numericLiteralMatcher,
	unaryOperatorMatcher,
	binaryOperatorMatcher,
	logicalOperatorMatcher,
	selectorMatcher,
	parenthesesMatcher,
	stringLiteralMatcher,
}

func expectOperand(cursor *parsly.Cursor) (ast.Node, error) {
	var err error
	pos := cursor.Pos
	match := cursor.MatchAfterOptional(whitespaceMatcher, operands...)
	switch match.Code {
	case boolLiteral:
		return &ast.Literal{Value: match.Text(cursor), Type: "bool"}, nil
	case stringLiteral:
		return &ast.Literal{Value: match.Text(cursor), Type: "string"}, nil
	case numericLiteral:
		return &ast.Literal{Value: match.Text(cursor), Type: "numeric"}, nil
	case singleQuotedStringLiteral:
		matched := match.Text(cursor)
		return &ast.Literal{Value: matched[1 : len(matched)-1], Type: "string", Quote: `'`}, nil
	case doubleQuotedStringLiteral:
		matched := match.Text(cursor)
		return &ast.Literal{Value: matched[1 : len(matched)-1], Type: "string", Quote: `"`}, nil
	case unaryOperator:
		op := match.Text(cursor)
		unary := &ast.Unary{Op: op}
		unary.X, err = expectOperand(cursor)
		if err != nil {
			return nil, err
		}
		return unary, nil
	case selectorCode:
		matched := match.Text(cursor)
		return &ast.Selector{X: matched}, nil
	case parenthesesCode:
		matched := match.Text(cursor)
		block := matched[1 : len(matched)-1]
		qualify, err := ParseCriteria(block)
		if err != nil || qualify == nil {
			return nil, err
		}
		return &ast.Group{X:qualify.X}, nil
	case parsly.EOF:
		return nil, nil
	case parsly.Invalid:
		return nil, cursor.NewError(operands...)
	default:
		cursor.Pos = pos
		return nil, nil
	}
}
