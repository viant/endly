package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkflowSelector(t *testing.T) {

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
		assert.EqualValues(t, useCase.ExpectedRelative, useCase.Selector.IsRelative(), "IsRelative() "+useCase.Description)

	}

}

func TestActionSelector(t *testing.T) {
	useCases := []struct {
		Description     string
		Selector        ActionSelector
		ExpectedService string
		ExpectedAction  string
	}{

		{
			Description:     "standard action selector with dot",
			Selector:        ActionSelector("exec.run"),
			ExpectedService: "exec",
			ExpectedAction:  "run",
		},
		{
			Description:     "standard action selector with colon",
			Selector:        ActionSelector("exec:run"),
			ExpectedService: "exec",
			ExpectedAction:  "run",
		},
		{
			Description:     "action without service",
			Selector:        ActionSelector("run"),
			ExpectedService: "workflow",
			ExpectedAction:  "run",
		},
		{
			Description:     "empty selector",
			Selector:        ActionSelector(""),
			ExpectedService: "workflow",
			ExpectedAction:  "",
		},
	}
	for _, useCase := range useCases {
		assert.EqualValues(t, useCase.ExpectedService, useCase.Selector.Service(), "Service() "+useCase.Description)
		assert.EqualValues(t, useCase.ExpectedAction, useCase.Selector.Action(), "Action() "+useCase.Description)
	}
}

func TestTasksSelector(t *testing.T) {
	useCases := []struct {
		Description string
		Selector    TasksSelector
		Expected    []string
		RunAll      bool
	}{

		{
			Description: "empty task selector",
			Selector:    TasksSelector(""),
			Expected:    []string{},
			RunAll:      true,
		},
		{
			Description: "wildcard task selector",
			Selector:    TasksSelector("*"),
			Expected:    []string{},
			RunAll:      true,
		},
		{
			Description: "single  task selector",
			Selector:    TasksSelector("task1"),
			Expected:    []string{"task1"},
			RunAll:      false,
		},

		{
			Description: "single  task selector",
			Selector:    TasksSelector("task1 , task3"),
			Expected:    []string{"task1", "task3"},
			RunAll:      false,
		},
	}
	for _, useCase := range useCases {
		assert.EqualValues(t, useCase.Expected, useCase.Selector.Tasks(), "Tasks() "+useCase.Description)
		assert.EqualValues(t, useCase.RunAll, useCase.Selector.RunAll(), "RunAll() "+useCase.Description)
	}
}
