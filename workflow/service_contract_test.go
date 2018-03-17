package workflow

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"testing"
)

func Test_NewPipelineRequestFromURL(t *testing.T) {

	var useCases = []struct {
		Description string
		URL         string
		Expected    interface{}
	}{
		{
			Description: "flat pipeline with action and workflow",
			URL:         "test/pipeline/flat.yaml",
			Expected: `[
  {
    "Name": "checkout",
    "Workflow": "",
    "Action": "vc:checkout",
    "Params": {
      "origin": [
        {
          "Key": "URL",
          "Value": "http://github.com/adrianwit/echo"
        }
      ],
      "secrets": [
        {
          "Key": "localhost",
          "Value": "localhsot"
        }
      ],
      "target": "URL:ssh://127.0.0.1 Credential:localhost"
    }
  },
  {
    "Name": "build",
    "Workflow": "docker/build:build",
    "Action": "",
    "Params": {
      "commands": [
        "apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
        "go get",
        "go version",
        "go build -a"
      ]
    }
  }
]`,
		},
		{
			Description: "nested pipeline with action and workflow",
			URL:         "test/pipeline/nested.yaml",
			Expected: `[
  {
    "Name": "system1",
    "Pipelines": [
      {
        "Name": "checkout",
        "Workflow": "",
        "Action": "vc:checkout",
        "Params": {
          "origin": [
            {
              "Key": "URL",
              "Value": "http://github.com/adrianwit/echo1"
            }
          ],
          "secrets": [
            {
              "Key": "localhost",
              "Value": "localhsot"
            }
          ],
          "target": "URL:ssh://127.0.0.1 Credential:localhost"
        }
      },
      {
        "Name": "build",
        "Workflow": "docker/build:build",
        "Action": "",
        "Params": {
          "commands": [
            "apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
            "go get",
            "go version",
            "go build -a"
          ]
        }
      }
    ]
  },
  {
    "Name": "system2",
    "Pipelines": [
      {
        "Name": "checkout",
        "Workflow": "",
        "Action": "vc:checkout",
        "Params": {
          "origin": [
            {
              "Key": "URL",
              "Value": "http://github.com/adrianwit/echo2"
            }
          ],
          "secrets": [
            {
              "Key": "localhost",
              "Value": "localhsot"
            }
          ],
          "target": "URL:ssh://127.0.0.1 Credential:localhost"
        }
      },
      {
        "Name": "build",
        "Workflow": "docker/build:build",
        "Action": "",
        "Params": {
          "commands": [
            "apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
            "go get",
            "go version",
            "go build -a"
          ]
        }
      }
    ]
  }
]`,
		},
	}

	for _, useCase := range useCases {
		request, err := NewPipelineRequestFromURL(useCase.URL)
		if assert.Nil(t, err, useCase.Description) {
			err = request.Init()
			if !assert.Nil(t, err, useCase.Description) {
				continue
			}
			assertly.AssertValues(t, useCase.Expected, request.Pipelines, useCase.Description)
		}
	}
}

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
	}

}

func TestActionSelector_Action(t *testing.T) {

	{
		action := ActionSelector("exec:run")
		assert.EqualValues(t, "run", action.Action())
	}
	{
		action := ActionSelector("exec")
		assert.EqualValues(t, "", action.Action())
	}

}

func TestActionSelector_Service(t *testing.T) {

	{
		action := ActionSelector("exec:run")
		assert.EqualValues(t, "exec", action.Service())
	}
	{
		action := ActionSelector("exec")
		assert.EqualValues(t, "exec", action.Service())
	}

}
