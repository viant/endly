package workflow

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkflowSelector_Name(t *testing.T) {

	useCases := []struct {
		Description      string
		Selector         WorkflowSelector
		ExpectedURL      string
		ExpectedName     string
		ExpectedTaks     string
		ExpectedRelative bool
	}{
		{
			Description:      "relative selector with wildcard",
			Selector:         "go/build.csv:*",
			ExpectedURL:      "go/build.csv",
			ExpectedName:     "build",
			ExpectedTaks:     "*",
			ExpectedRelative: true,
		},
		{
			Description:      "relative selector with tasks",
			Selector:         "go/build.csv:build",
			ExpectedURL:      "go/build.csv",
			ExpectedName:     "build",
			ExpectedTaks:     "build",
			ExpectedRelative: true,
		},
		{
			Description:      "relative selector without tasks",
			ExpectedURL:      "build.csv",
			Selector:         "build",
			ExpectedName:     "build",
			ExpectedTaks:     "*",
			ExpectedRelative: true,
		},
		{
			Description:      "absolute URL selector without tasks",
			ExpectedURL:      "http://abc.com/path/build.csv",
			Selector:         "http://abc.com/path/build",
			ExpectedName:     "build",
			ExpectedTaks:     "*",
			ExpectedRelative: false,
		},
		{
			Description:      "absolute URL selector with tasks",
			Selector:         "http://abc.com/path/build:task1",
			ExpectedURL:      "http://abc.com/path/build.csv",
			ExpectedName:     "build",
			ExpectedTaks:     "task1",
			ExpectedRelative: false,
		},
	}

	for _, useCase := range useCases {
		assert.EqualValues(t, useCase.ExpectedURL, useCase.Selector.URL(), "URL() "+useCase.Description)
		assert.EqualValues(t, useCase.ExpectedName, useCase.Selector.Name(), "Name() "+useCase.Description)
		assert.EqualValues(t, useCase.ExpectedTaks, useCase.Selector.Tasks(), "Task() "+useCase.Description)
	}

}
