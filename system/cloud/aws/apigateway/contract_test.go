package apigateway

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"testing"
)

func TestSetupResourceInput_ParentPath(t *testing.T) {

	var useCases =[]struct{
		description string
		path string
		pathPart interface{}
		expected string
	}{
		{
			description:"simply hierarchy",
			path:"/v1/api",
			pathPart:"api",
			expected:"/v1/",
		},
		{
			description:"simply hierarchy expr",
			path:"/v1/api/entity/${id}",
			pathPart:"entity/${id}",
			expected:"/v1/api/",
		},

	}


	for _, useCase := range useCases {
		input := &SetupResourceInput{}
		input.Path = useCase.path
		if useCase.pathPart !=nil {
			pathPart := toolbox.AsString(useCase.pathPart)
			input.PathPart = &pathPart
		}
		actual := input.ParentPath()
		assert.EqualValues(t, useCase.expected, actual, useCase.description)
	}



}