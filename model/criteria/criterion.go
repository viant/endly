package criteria

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

// Criterion represent evaluation criterion
type Criterion struct {
	*Predicate
	LeftOperand  interface{}
	Operator     string
	RightOperand interface{}
}

func (c *Criterion) expandOperand(opperand interface{}, state data.Map) interface{} {
	if opperand == nil {
		return nil
	}
	return state.Expand(opperand)
}

func checkUndefined(err error, left, right interface{}, operator string) error {
	if err != nil {
		if text, ok := left.(string); ok {
			return fmt.Errorf("undefined %v  in expression: %v %s %v", text, left, operator, right)
		}
		if text, ok := right.(string); ok {
			return fmt.Errorf("undefined %v  in expression: %v %s %v", text, left, operator, right)
		}

	}
	return err
}

// Apply evaluates criterion with supplied context and state map . Dolar prefixed $expression will be expanded before evaluation.
func (c *Criterion) Apply(state data.Map) (bool, error) {
	if c.Predicate != nil && len(c.Predicate.Criteria) > 0 {
		return c.Predicate.Apply(state)
	}
	leftOperand := c.expandOperand(c.LeftOperand, state)
	rightOperand := c.expandOperand(c.RightOperand, state)
	var err error
	var leftNumber, rightNumber float64
	var rootPath = assertly.NewDataPath("/")
	var context = assertly.NewDefaultContext()

	if text, ok := leftOperand.(string); ok {
		switch text {
		case "t", "T", "true", "TRUE", "True":
			leftOperand = true
		case "f", "F", "false", "FALSE", "False":
			leftOperand = false
		}
	}

	if rightOperand == nil {
		switch leftOperand.(type) {
		case bool:
			rightOperand = false
		case string:
			rightOperand = ""
		}
	}
	switch c.Operator {
	case "=", ":":
		validation, err := assertly.AssertWithContext(rightOperand, leftOperand, rootPath, context)
		if err != nil {
			return false, err
		}
		return validation.FailedCount == 0, nil
	case "!=", "":
		validation, err := assertly.AssertWithContext(leftOperand, rightOperand, rootPath, context)
		if err != nil {
			return false, err
		}
		return validation.FailedCount > 0, nil
	case ">=":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber >= rightNumber, nil
			}
		}
	case "<=":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber <= rightNumber, nil
			}
		}

	case ">":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber > rightNumber, nil
			}
		}
	case "<":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber < rightNumber, nil
			}
		}
	}
	err = checkUndefined(err, leftOperand, rightNumber, c.Operator)
	return false, err
}

// NewCriterion creates a new criterion
func NewCriterion(leftOperand interface{}, operator string, rightOperand interface{}) *Criterion {
	return &Criterion{
		LeftOperand:  leftOperand,
		Operator:     operator,
		RightOperand: rightOperand,
	}
}
