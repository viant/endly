package selenium

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
)

func TestParser_Parse(t *testing.T) {

	var useCases = []struct {
		Description string
		Command     string
		Expected    *Action
		HasError    bool
	}{
		{
			Description: "WebDriver call",
			Command:     "get(http://127.0.0.1:8888/signup/)",
			Expected:    NewAction("", "", "Get", "http://127.0.0.1:8888/signup/"),
		},
		{
			Description: "WebDriver call empty",
			Command:     "get",
			Expected:    NewAction("", "", "Get", ),
		},
		{
			Description: "WebDriver call assigments",
			Command:     "key1 = get(http://127.0.0.1:8888/signup/)",
			Expected:    NewAction("key1", "", "Get", "http://127.0.0.1:8888/signup/"),
		},


		{
			Description: "WebElement call",
			Command:     "(#email).clear",
			Expected:    NewAction("", "#email", "Clear"),
		},

		{
			Description: "xpath selector WebElement call",
			Command:     "(xpath://SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]).text",
			Expected:    NewAction("", "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]", "Text"),
		},

		{
			Description: "xpath selector WebElement call assigment",
			Command:     "key1 = (xpath://SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]).text",
			Expected:    NewAction("key1", "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]", "Text"),
		},

	}

	parser := &parser{}
	for _, useCase := range useCases {

		action, err := parser.Parse(useCase.Command)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if ! assert.Nil(t, err, useCase.Description) {
			continue
		}
		assertly.AssertValues(t, useCase.Expected, action, useCase.Description)

	}

}
