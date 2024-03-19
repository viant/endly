package compiler

import (
	"fmt"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

type binary struct {
	x, y *operand
}

func NewBinary(op string, operands ...*Operand) New {
	switch op {
	case "=", "==", ":", ":!", "<>", "!=", "<", ">", "<=", ">=", "contains", "contains!", "&&", "||":
	default:
		return func() (eval.Compute, error) {
			return nil, fmt.Errorf("unsupported operator: %v", op)
		}
	}

	return func() (eval.Compute, error) {
		expr := &binary{
			x: operands[0].operand(),
			y: operands[1].operand(),
		}
		switch op {
		case ":", "=", "==":
			return expr.equal, nil
		case "<>", ":!", "!=":
			return expr.notEqual, nil
		case "<":
			return expr.lessThan, nil
		case ">":
			return expr.greaterThan, nil
		case "<=":
			return expr.lessThanEqual, nil
		case ">=":
			return expr.greaterThanEqual, nil
		case "contains":
			return expr.contains, nil
		case "contains!":
			return expr.notContains, nil
		case "&&":
			return expr.and, nil
		case "||":
			return expr.or, nil
		default:
			return nil, fmt.Errorf("unsupported operator: %v", op)
		}
	}
}
func (b *binary) xValue(state data.Map) (interface{}, bool, error) {
	return b.x.Value(state)
}

func (b *binary) yValue(state data.Map) (interface{}, bool, error) {
	return b.y.Value(state)
}

func (b *binary) equal(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return nil, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return nil, false, errY
	}
	switch x.(type) {
	case string:
		y = toolbox.AsString(y)
	case int:
		y = toolbox.AsInt(y)
	case float64:
		y = toolbox.AsFloat(y)
	case bool:
		y = toolbox.AsBoolean(y)

	}

	if !hasX && !hasY {
		return false, false, nil
	}
	if hasX != hasY {
		return false, true, nil
	}
	return x == y, true, nil
}

func (b *binary) notEqual(state data.Map) (interface{}, bool, error) {
	ret, has, err := b.equal(state)
	if err != nil {
		return false, false, err
	}
	eval := false
	ok := false
	if has {
		eval, ok = ret.(bool)
		if !ok {
			return nil, false, fmt.Errorf("expected boolean but had: %T", ret)
		}
		return !eval, has, nil
	}
	return eval, has, nil
}

func (b *binary) lessThan(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xNum := toolbox.AsFloat(x)
	yNum := toolbox.AsFloat(y)
	return xNum < yNum, true, nil
}

func (b *binary) lessThanEqual(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xNum := toolbox.AsFloat(x)
	yNum := toolbox.AsFloat(y)
	return xNum <= yNum, true, nil

}

func (b *binary) greaterThanEqual(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xNum := toolbox.AsFloat(x)
	yNum := toolbox.AsFloat(y)
	return xNum >= yNum, true, nil
}

func (b *binary) greaterThan(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return nil, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return nil, false, errY
	}

	if !hasX && !hasY {
		return nil, false, nil
	}
	xNum := toolbox.AsFloat(x)
	yNum := toolbox.AsFloat(y)
	return xNum > yNum, true, nil
}

func (b *binary) contains(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xText := toolbox.AsString(x)
	yText := toolbox.AsString(y)
	return strings.Contains(xText, yText), true, nil
}

func (b *binary) notContains(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xText := toolbox.AsString(x)
	yText := toolbox.AsString(y)
	return !strings.Contains(xText, yText), true, nil
}

func (b *binary) and(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX

	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}

	if !hasX && !hasY {
		return false, false, nil
	}
	xBool := toolbox.AsBoolean(x)
	if ! xBool {
		return false, true, nil
	}
	yBool := toolbox.AsBoolean(y)
	return xBool && yBool, true, nil
}

func (b *binary) or(state data.Map) (interface{}, bool, error) {
	x, hasX, errX := b.xValue(state)
	if errX != nil {
		return false, false, errX
	}

	xBool := toolbox.AsBoolean(x)
	if xBool {
		return true, true, nil
	}
	y, hasY, errY := b.yValue(state)
	if errY != nil {
		return false, false, errY
	}
	if !hasX && !hasY {
		return false, false, nil
	}

	yBool := toolbox.AsBoolean(y)
	return xBool || yBool, true, nil
}
