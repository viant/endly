package compiler

import (
	"fmt"
	"github.com/viant/endly/model/criteria/ast"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/endly/model/criteria/parser"
)

//New represents a new evaluator
type New func() (eval.Compute, error)

func Compile(expr string) (New, error) {
	node, err := parser.ParseCriteria(expr)
	if err != nil {
		return nil, err
	}
	return compile(node)
}

func compile(node ast.Node) (New, error) {
	switch actual := node.(type) {
	case *ast.Binary:
		x, xErr := NewOperand(actual.X)
		if xErr != nil {
			return nil, xErr
		}
		y, yErr := NewOperand(actual.Y)
		if yErr != nil {
			return nil, yErr
		}
		return NewBinary(actual.Op, x, y), nil
	case *ast.Unary:
		x, xErr := NewOperand(actual.X)
		if xErr != nil {
			return nil, xErr
		}
		return NewUnary(actual.Op, x), nil
	case *ast.Qualify:
		return compile(actual.X)
	case *ast.Group:
		return compile(actual.X)
	default:
		if node == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("unsupported node type: %T", actual)
	}
}

func NewOperand(x ast.Node) (*Operand, error) {
	switch actual := x.(type) {
	case *ast.Literal:
		return &Operand{literal: actual}, nil
	case *ast.Selector:
		return &Operand{selector: actual}, nil
	default:
		if x == nil {
			return &Operand{nil: true}, nil
		}
		compute, err := compile(actual)
		if err != nil {
			return nil, err
		}
		return &Operand{compute: compute}, nil
	}
}
