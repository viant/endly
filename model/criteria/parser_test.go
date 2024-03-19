package criteria_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly/model/criteria/compiler"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestCriteriaParser_Parse(t *testing.T) {

	var useCases = []struct {
		description string
		state       data.Map
		expression string
		expect bool
		hasError   bool
	}{

		{
			description: "Unicode operator criterion",
			expression:  "$counter \u003e 10",
			state: map[string]interface{}{
				"counter":11,
			},
			expect: true,
		},
		{
			description: "Empty left operand criterion",
			expression:  ":!$value",
			expect: false,
		},
		{
			description: "UDFs criterion",
			expression:  "$HasResource(${buildHost}${buildDirectory}/pom.xml):false",
			expect: false,
		},

		{
			description: "Simple criterion",
			expression:  "$key1 = 123",
			state: map[string]interface{}{
				"key1":123,
			},
			expect: true,
		},
		{
			description: "AND criteria",
			expression:  "$key1 = 123 && $key2 > 12",
			state: map[string]interface{}{
				"key1":123,
				"key2":13,
			},
			expect: true,
		},
		{
			description: "AND criteria",
			expression:  "$key1 = 123 && $key2 > 12",
			state: map[string]interface{}{
				"key1":123,
				"key2":11,
			},
			expect: false,
		},
		{
			description: "OR criteria - true",
			expression:  "($key1 = 123 && $key2 > 12) || $k3 contains 123 || $z",
			state: map[string]interface{}{
				"k3":"123",
			},
			expect: true,
		},
		{
			description: "OR criteria - false",
			expression:  "($key1 = 123 && $key2 > 12) || $k3 contains 123 || $z",
			state: map[string]interface{}{
				"k3":"1",
			},
			expect: false,
		},
	}

	for _, useCase := range useCases {
		newCompute, err := compiler.Compile(useCase.expression)
		if ! assert.Nil(t, err, useCase.description) {
			continue
		}
		compute, _ := newCompute()
		actual, _, err := compute(useCase.state)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		assert.Nil(t, err, useCase.description)
		assertly.AssertValues(t, useCase.expect, actual, useCase.description)
	}

}
