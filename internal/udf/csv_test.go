package udf

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewCsvReader(t *testing.T) {

	resultMap := map[string]interface{}{
		"name":          "Bob",
		"facoriteColor": "red",
	}

	rBytes, _ := json.Marshal(resultMap)

	fmt.Printf("!%v!", string(rBytes))

	useCases := []struct {
		description string
		header      string
		delimiter   string
		data        string
		expected    string
		hasError    bool
	}{
		{
			description: "simple csv",
			header:      "first_name,last_name,username",
			data: `"Rob","Pike",rob
Ken,Thompson,ken
`,
			expected: `[{"first_name":"Rob"},{"first_name":"Ken"}]`,
		},
		{
			description: "no header",
			header:      "",
			hasError:    true,
		},
		{
			description: "invalid header or delimiter",
			header:      "id;name",
			hasError:    true,
		},
	}
	for _, useCase := range useCases {
		udfFunc, err := NewCsvReader(useCase.header, useCase.delimiter)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err) {
			continue
		}
		transformed, err := udfFunc(useCase.data, nil)
		if !assert.Nil(t, err) {
			continue
		}
		actual := toolbox.AsString(transformed)
		assertly.AssertValues(t, useCase.expected, actual, useCase.description)
	}

}
