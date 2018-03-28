package selenium

import (
	"github.com/stretchr/testify/assert"
	"testing"
)



func TestWebSelector_ByAndValue(t *testing.T) {

	var useCases = []struct {
		Description   string
		Selector      string
		ExpectedBy    string
		ExpectedValue string
	}{
		{
			Description:   "css selector",
			Selector:      "#id",
			ExpectedBy:    "css selector",
			ExpectedValue: "#id",
		},
		{
			Description:   "class selector",
			Selector:      ".red",
			ExpectedBy:    "class name",
			ExpectedValue: ".red",
		},
		{
			Description:   "tag selector",
			Selector:      "li",
			ExpectedBy:    "tag name",
			ExpectedValue: "li",
		},

		{
			Description:   "xpath selector",
			Selector:      "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]",
			ExpectedBy:    "xpath",
			ExpectedValue: "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]",
		},
	}

	for _, useCase := range useCases {
		selector := WebSelector(useCase.Selector)
		by, value := selector.ByAndValue()
		assert.EqualValues(t, useCase.ExpectedBy, by, "By "+useCase.Description)
		assert.EqualValues(t, useCase.ExpectedValue, value, "Value "+useCase.Description)
	}

}


func TestWebElementSelector_Init(t *testing.T) {



	var elem =  NewWebElementSelector("", "#name")
	assert.Nil(t, elem.Init())
	assert.Nil(t, elem.Validate())
	assert.EqualValues(t, "css selector", elem.By)
	assert.EqualValues(t, "#name", elem.Value)

}

