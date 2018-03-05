package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"testing"
)

func TestCriteriaParser_Parse(t *testing.T) {

	parser := endly.NewCriteriaParser()

	var useCases = []struct {
		Description string
		Expression  string
		Expected    *endly.Criteria
		HasError    bool
	}{

		{
			Description: "Empty left operand criterion",
			Expression:  ":!$value",
			Expected:    endly.NewCriteria("", endly.NewCriterion(nil, ":", "!$value")),
		},
		{
			Description: "UDF criterion",
			Expression:  "$HasResource(${buildHost}${buildDirectory}/pom.xml):false",
			Expected:    endly.NewCriteria("", endly.NewCriterion("$HasResource(${buildHost}${buildDirectory}/pom.xml)", ":", "false")),
		},


		{
			Description: "Simple criterion",
			Expression:  "$key1 = 123",
			Expected:    endly.NewCriteria("", endly.NewCriterion("$key1", "=", "123")),
		},
		{
			Description: "AND criteria",
			Expression:  "$key1 = 123 && $key2 > 12",
			Expected:    endly.NewCriteria("&&", endly.NewCriterion("$key1", "=", "123"), endly.NewCriterion("$key2", ">", "12")),
		},
		{
			Description: "OR criteria",
			Expression:  "$key1 = 123 && $key2 > 12 || $k3: /123/ || $z",
			Expected: endly.NewCriteria("&&",
				endly.NewCriterion("$key1", "=", "123"),
				endly.NewCriterion("$key2", ">", "12"),
				&endly.Criterion{
					Criteria: endly.NewCriteria("||",
						endly.NewCriterion("$k3", ":", "/123/"),
						endly.NewCriterion("$z", "", nil)),
				}),
		},
		{
			Description: "Grouping criterion",
			Expression:  "$k0 && ($k1 || $k2)",
			Expected: endly.NewCriteria("&&",
				endly.NewCriterion("$k0", "", nil),
				&endly.Criterion{
					Criteria: endly.NewCriteria("||",
						endly.NewCriterion("$k1", "", nil),
						endly.NewCriterion("$k2", "", nil)),
				},
			),
		},
		{
			Description: "assertly criterion",
			Expression:  "$key1 : 123 3",
			Expected:    endly.NewCriteria("", endly.NewCriterion("$key1", ":", "123 3")),
		},
	}

	for _, useCase := range useCases {
		criteria, err := parser.Parse(useCase.Expression)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.Nil(t, err, useCase.Description)
		assertly.AssertValues(t, useCase.Expected, criteria)
	}

}
