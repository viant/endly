package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestCriterion_IsTrue(t *testing.T) {

	useCases := []struct {
		Description string
		Criterion   *endly.Criterion
		HasError    bool
		IsTrue      bool
	}{

		{
			Description: "error test",
			Criterion:   endly.NewCriterion("123", ":", "/[12.13]/"),
			HasError:    true,
		},
		{
			Description: "grater than test ",
			Criterion:   endly.NewCriterion("21", ">", "10"),
			IsTrue:      true,
		},

		{
			Description: "equal test",
			Criterion:   endly.NewCriterion("12", "", "12"),
			IsTrue:      true,
		},
		{
			Description: ">= test",
			Criterion:   endly.NewCriterion("12", ">=", "12"),
			IsTrue:      true,
		},
		{
			Description: "> test",
			Criterion:   endly.NewCriterion("12", ">", "12"),
			IsTrue:      false,
		},
		{
			Description: "<= test",
			Criterion:   endly.NewCriterion("12", "<=", "12"),
			IsTrue:      true,
		},
		{
			Description: "< test",
			Criterion:   endly.NewCriterion("12", "<", "12"),
			IsTrue:      false,
		},
		{
			Description: "!= test",
			Criterion:   endly.NewCriterion("12", "!=", "12"),
			IsTrue:      false,
		},
		{
			Description: "assertly test",
			Criterion:   endly.NewCriterion("12", ":", "12"),
			IsTrue:      true,
		},
	}

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	for _, useCase := range useCases {
		isTrue, err := useCase.Criterion.IsTrue(context, context.State())
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
		Criteria    *endly.Criteria
		HasError    bool
		IsTrue      bool
	}{
		{
			Description: "OR true test ",
			Criteria: endly.NewCriteria("||",
				endly.NewCriterion("1", "=", "1"),
				endly.NewCriterion("2", ">", "2")),
			IsTrue: true,
		},
		{
			Description: "OR false test",
			Criteria: endly.NewCriteria("||",
				endly.NewCriterion("10", "=", "1"),
				endly.NewCriterion("2", ">", "2")),
			IsTrue: false,
		},
		{
			Description: "AND true test",
			Criteria: endly.NewCriteria("&&",
				endly.NewCriterion("1", "=", "1"),
				endly.NewCriterion("2", "=", "2")),
			IsTrue: true,
		},
		{
			Description: "AND false test",
			Criteria: endly.NewCriteria("&&",
				endly.NewCriterion("1", "<", "1"),
				endly.NewCriterion("2", "=", "2")),
			IsTrue: false,
		},
	}

	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	for _, useCase := range useCases {
		isTrue, err := useCase.Criteria.IsTrue(context, context.State())
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.EqualValues(t, useCase.IsTrue, isTrue, useCase.Description)
	}
}
