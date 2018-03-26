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
			Description: "nested data pipeline",
			URL:         "test/pipeline/data.yaml",
			Expected: `[
  {
    "Name": "create-db",
    "Workflow": "",
    "Action": "dsunit:register",
    "Params": {
      "admin": {
        "config": {
          "credentials": "$mysqlCredential",
          "descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
          "driverName": "mysql"
        },
        "datastore": "mysql"
      },
      "config": {
        "credentials": "$mysqlCredential",
        "descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
        "driverName": "mysql"
      },
      "datastore": "db1",
      "recreate": true,
      "scripts": [
        {
          "URL": "data/db1/schema.ddl"
        }
      ]
    }
  },
  {
    "Name": "populate",
    "Workflow": "",
    "Action": "dsunit:prepare",
    "Params": {
      "URL": "datastore/db1/dictionary",
      "datastore": "db1"
    }
  }
]`,
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
         "commands": [
        "apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
        "go get",
        "go version",
        "go build -a"
      ],
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
            "commands": [
		   "apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
           "go get",
           "go version",
           "go build -a"
			],
            "secrets": {
              "localhost": "localhsot"
            },
            "target": "URL:ssh://127.0.0.1 Credentials:localhost"
          },
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
            "commands": [
   			"apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
            "go get",
            "go version",
            "go build -a"
			],
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

	for _, useCase := range useCases {

		request, err := NewRunRequestFromURL(useCase.URL)
		if assert.Nil(t, err, useCase.Description) {
			err = request.Init()
			if !assert.Nil(t, err, useCase.Description) {
				t.Log(err)
				continue
			}
			assertly.AssertValues(t, useCase.Expected, request.Pipelines, useCase.Description)
		}
	}
}
