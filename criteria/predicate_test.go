package criteria_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/toolbox"
	"testing"
)

func TestPredicate_Apply(t *testing.T) {
	useCases := []struct {
		Description string
		Criteria    *criteria.Predicate
		HasError    bool
		IsTrue      bool
	}{
		{
			Description: "OR true test ",
			Criteria: criteria.NewPredicate("||",
				criteria.NewCriterion("1", "=", "1"),
				criteria.NewCriterion("2", ">", "2")),
			IsTrue: true,
		},
		{
			Description: "OR false test",
			Criteria: criteria.NewPredicate("||",
				criteria.NewCriterion("10", "=", "1"),
				criteria.NewCriterion("2", ">", "2")),
			IsTrue: false,
		},
		{
			Description: "AND true test",
			Criteria: criteria.NewPredicate("&&",
				criteria.NewCriterion("1", "=", "1"),
				criteria.NewCriterion("2", "=", "2")),
			IsTrue: true,
		},
		{
			Description: "AND false test",
			Criteria: criteria.NewPredicate("&&",
				criteria.NewCriterion("1", "<", "1"),
				criteria.NewCriterion("2", "=", "2")),
			IsTrue: false,
		},
	}

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	for _, useCase := range useCases {
		isTrue, err := useCase.Criteria.Apply(context.State())
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.IsTrue, isTrue, useCase.Description)
	}
}
