package criteria_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/toolbox"
	"testing"
)

func TestCriterion_IsTrue(t *testing.T) {

	useCases := []struct {
		Description string
		Criterion   *criteria.Criterion
		HasError    bool
		IsTrue      bool
	}{

		{
			Description: "error test",
			Criterion:   criteria.NewCriterion("123", ":", "/[12.13]/"),
			HasError:    true,
		},
		{
			Description: "grater than test ",
			Criterion:   criteria.NewCriterion("21", ">", "10"),
			IsTrue:      true,
		},

		{
			Description: "equal test",
			Criterion:   criteria.NewCriterion("12", ":", "12"),
			IsTrue:      true,
		},
		{
			Description: ">= test",
			Criterion:   criteria.NewCriterion("12", ">=", "12"),
			IsTrue:      true,
		},
		{
			Description: "> test",
			Criterion:   criteria.NewCriterion("12", ">", "12"),
			IsTrue:      false,
		},
		{
			Description: "<= test",
			Criterion:   criteria.NewCriterion("12", "<=", "12"),
			IsTrue:      true,
		},
		{
			Description: "< test",
			Criterion:   criteria.NewCriterion("12", "<", "12"),
			IsTrue:      false,
		},
		{
			Description: "!= test",
			Criterion:   criteria.NewCriterion("12", "!=", "12"),
			IsTrue:      false,
		},
		{
			Description: "assertly test",
			Criterion:   criteria.NewCriterion("12", ":", "12"),
			IsTrue:      true,
		},

		{
			Description: "left operand criteria only",
			Criterion:   criteria.NewCriterion("$ok", "", nil),
			IsTrue:      true,
		},
		{
			Description: "empty left operand criteria only",
			Criterion:   criteria.NewCriterion("", "", nil),
			IsTrue:      false,
		},
	}

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	for _, useCase := range useCases {
		isTrue, err := useCase.Criterion.IsTrue(context.State())
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.IsTrue, isTrue, useCase.Description)
	}
}

func TestCriteria_IsTrue(t *testing.T) {
	useCases := []struct {
		Description string
		Criteria    *criteria.Criteria
		HasError    bool
		IsTrue      bool
	}{
		{
			Description: "OR true test ",
			Criteria: criteria.NewCriteria("||",
				criteria.NewCriterion("1", "=", "1"),
				criteria.NewCriterion("2", ">", "2")),
			IsTrue: true,
		},
		{
			Description: "OR false test",
			Criteria: criteria.NewCriteria("||",
				criteria.NewCriterion("10", "=", "1"),
				criteria.NewCriterion("2", ">", "2")),
			IsTrue: false,
		},
		{
			Description: "AND true test",
			Criteria: criteria.NewCriteria("&&",
				criteria.NewCriterion("1", "=", "1"),
				criteria.NewCriterion("2", "=", "2")),
			IsTrue: true,
		},
		{
			Description: "AND false test",
			Criteria: criteria.NewCriteria("&&",
				criteria.NewCriterion("1", "<", "1"),
				criteria.NewCriterion("2", "=", "2")),
			IsTrue: false,
		},
	}

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	for _, useCase := range useCases {
		isTrue, err := useCase.Criteria.IsTrue(context.State())
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.IsTrue, isTrue, useCase.Description)
	}
}
