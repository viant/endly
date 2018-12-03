package criteria_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly/criteria"
	"testing"
)

func TestCriteriaParser_Parse(t *testing.T) {

	parser := criteria.NewParser()

	var useCases = []struct {
		Description string
		Expression  string
		Expected    *criteria.Predicate
		HasError    bool
	}{

		{
			Description: "Unicode operator criterion",
			Expression:  "$counter \u003e 10",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$counter", ">", "10")),
		},
		{
			Description: "Empty left operand criterion",
			Expression:  ":!$value",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion(nil, ":", "!$value")),
		},
		{
			Description: "UDFs criterion",
			Expression:  "$HasResource(${buildHost}${buildDirectory}/pom.xml):false",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$HasResource(${buildHost}${buildDirectory}/pom.xml)", ":", "false")),
		},

		{
			Description: "Simple criterion",
			Expression:  "$key1 = 123",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$key1", "=", "123")),
		},
		{
			Description: "AND criteria",
			Expression:  "$key1 = 123 && $key2 > 12",
			Expected:    criteria.NewPredicate("&&", criteria.NewCriterion("$key1", "=", "123"), criteria.NewCriterion("$key2", ">", "12")),
		},
		{
			Description: "OR criteria",
			Expression:  "$key1 = 123 && $key2 > 12 || $k3: /123/ || $z",
			Expected: criteria.NewPredicate("&&",
				criteria.NewCriterion("$key1", "=", "123"),
				criteria.NewCriterion("$key2", ">", "12"),
				&criteria.Criterion{
					Predicate: criteria.NewPredicate("||",
						criteria.NewCriterion("$k3", ":", "/123/"),
						criteria.NewCriterion("$z", "", nil)),
				}),
		},
		{
			Description: "Grouping criterion",
			Expression:  "$k0 && ($k1 || $k2)",
			Expected: criteria.NewPredicate("&&",
				criteria.NewCriterion("$k0", "", nil),
				&criteria.Criterion{
					Predicate: criteria.NewPredicate("||",
						criteria.NewCriterion("$k1", "", nil),
						criteria.NewCriterion("$k2", "", nil)),
				},
			),
		},
		{
			Description: "assertly criterion",
			Expression:  "$key1 : 123 3",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$key1", ":", "123 3")),
		},
		{
			Description: "left operand criterion",
			Expression:  "$key1",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$key1", "", nil)),
		},
		{
			Description: "expected logical conjunction patch",
			Expression:  "$stdout :/(END)/",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$stdout", ":", "/(END)/")),
		},

		{
			Description: "quoted operand criterion",
			Expression:  "$key1 = 'abc'",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$key1", "=", "abc")),
		},

		{
			Description: "uni operan boolean expression",
			Expression:  "$key1",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$key1", "!=", nil)),
		},
		{
			Description: "uni operand boolean expression",
			Expression:  "$HasResource(file:///tmp/req/print.json)",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$HasResource(file:///tmp/req/print.json)", "!=", nil)),
		},

		{
			Description: "udf usage",
			Expression:  "$i < $Len($params.requests)",
			Expected:    criteria.NewPredicate("", criteria.NewCriterion("$i", "<", "$Len($params.requests)")),

		},

		//$stdout :/(END)/
	}

	for _, useCase := range useCases {
		predicate, err := parser.Parse(useCase.Expression)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		assert.Nil(t, err, useCase.Description)
		assertly.AssertValues(t, useCase.Expected, predicate)
	}

}
