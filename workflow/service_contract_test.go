package workflow

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"testing"
	"github.com/viant/toolbox"
	"fmt"
)

func Test_NewPipelineRequestFromURL(t *testing.T) {

	var useCases = []struct {
		Description string
		URL         string
		Expected    interface{}
	}{

		{
			Description: "nested data pipeline",
			URL:         "test/pipeline/data.yaml",
			Expected: `[`,
		},

		{
			Description: "flat pipeline with action and workflow",
			URL:         "test/pipeline/flat.yaml",
			Expected: `[
    {
      "Name": "checkout",
      "Action": "vc:checkout",
      "Params": {
        "origin": {
          "URL": "http://github.com/adrianwit/echo"
        },
        "secrets": {
          "localhost": "localhsot"
        },
        "target": {
          "Credentials": "localhost",
          "URL": "ssh://127.0.0.1"
        }
      }
    },
    {
      "Name": "build",
      "Workflow": "docker/build:build",
      "Params": {
        "commands": {},
        "secrets": {
          "localhost": "localhsot"
        },
        "target": {
          "Credentials": "localhost",
          "URL": "ssh://127.0.0.1"
        }
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
          "Action": "vc:checkout",
          "Params": {
            "origin": {
              "URL": "http://github.com/adrianwit/echo1"
            },
            "secrets": {
              "localhost": "localhsot"
            },
            "target": "URL:ssh://127.0.0.1 Credentials:localhost"
          }
        },
        {
          "Name": "build",
          "Workflow": "docker/build:build",
          "Params": {
            "commands": {},
            "secrets": {
              "localhost": "localhsot"
            },
            "target": "URL:ssh://127.0.0.1 Credentials:localhost"
          },
          "When": "",
          "Pipelines": []
        }
      ]
    },
    {
      "Name": "system2",
      "Pipelines": [
        {
          "Name": "checkout",
          "Action": "vc:checkout",
          "Params": {
            "origin": {
              "URL": "http://github.com/adrianwit/echo2"
            },
            "secrets": {
              "localhost": "localhsot"
            },
            "target": "URL:ssh://127.0.0.1 Credentials:localhost"
          }
        },
        {
          "Name": "build",
          "Workflow": "docker/build:build",
          "Params": {
            "commands": {},
            "secrets": {
              "localhost": "localhsot"
            },
            "target": "URL:ssh://127.0.0.1 Credentials:localhost"
          }
        }
      ]
    }
  ]`,
		},
	}

	for i, useCase := range useCases {
		if i != 0 {
			continue
		}
		request, err := NewRunRequestFromURL(useCase.URL)
		if assert.Nil(t, err, useCase.Description) {
			err = request.Init()
			if !assert.Nil(t, err, useCase.Description) {
				continue
			}

			rr, _ := toolbox.AsJSONText(request)
			fmt.Printf("%v\n", rr)

			assertly.AssertValues(t, useCase.Expected, request.Pipelines, useCase.Description)
		}
	}
}
