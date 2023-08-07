package criteria

import (
	"github.com/viant/toolbox/data"
)

// Predicate represents logical criteria
type Predicate struct {
	LogicalOperator string
	Criteria        []*Criterion
}

// Apply evaluates criteria with supplied context and state map . Dolar prefixed $expression will be expanded before evaluation.
func (c *Predicate) Apply(state data.Map) (bool, error) {
	if c.LogicalOperator == "||" {
		for _, criterion := range c.Criteria {
			result, err := criterion.Apply(state)
			if result || err != nil {
				return result, err
			}
		}
		return false, nil
	}
	for _, criterion := range c.Criteria {
		result, err := criterion.Apply(state)
		if !result || err != nil {
			return result, err
		}
	}
	return true, nil
}

// NewPredicate creates a new criteria for supplied logical operator and criteria
func NewPredicate(operator string, criteria ...*Criterion) *Predicate {
	return &Predicate{
		LogicalOperator: operator,
		Criteria:        criteria,
	}
}
