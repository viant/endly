package compiler

import (
	"fmt"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

func NewUnary(op string, operand *Operand) New {
	switch op {
	case "!", "not":
		return func() (eval.Compute, error) {
			op := operand.operand()
			return func(state data.Map) (interface{}, bool, error) {
				value, ok, err := op.Value(state)
				if err != nil {
					return nil, false, err
				}
				return !toolbox.AsBoolean(value), ok, nil
			}, nil
		}

	case "":
		return func() (eval.Compute, error) {
			op := operand.operand()
			return func(state data.Map) (interface{}, bool, error) {
				value, ok, err := op.Value(state)
				if err != nil {
					return nil, false, err
				}
				if !ok {
					return false, true, nil
				}
				switch actual := value.(type) {
				case string:
					switch strings.TrimSpace(actual) {
					case "false", "0", "":
						return false, true, nil
					}
					return true, true, nil
				case bool:
					return actual, true, nil
				case int:
					return actual != 0, true, nil
				case float64:
					return actual != 0, true, nil
				}
				return value != nil, ok, nil
			}, nil
		}
	case "defined":
		return func() (eval.Compute, error) {
			op := operand.operand()
			return func(state data.Map) (interface{}, bool, error) {
				_, ok, err := op.Value(state)
				if err != nil {
					return nil, false, err
				}
				return ok, true, nil
			}, nil
		}

	default:
		return func() (eval.Compute, error) {
			return nil, fmt.Errorf("unsupported unary operator: %v", op)
		}
	}
}
