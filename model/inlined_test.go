package model

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestPipelines_AsWorkflow(t *testing.T) {

	useCases := []struct {
		Description string
		YAMLData    string
		Expected    interface{}
		HasError    bool
	}{
		{
			Description: "workflow/action pipeline",
			YAMLData: `pipeline:
  mysql:
    workflow: service/mysql:start
  catch:
    action: print
    message: error
 `,
			Expected: `{
	"OnErrorTask": "catch",
	"tasks": [
		{
			"Actions": [
				{
					"Action": "run",
					"Name": "mysql",
					"Repeat": 1,
					"Request": {
						"URL": "service/mysql.csv",
						"tasks": "start"
					},
					"Service": "workflow",
					"Tag": "mysql",
					"TagID": "mysql"
				}
			],
			"Name": "mysql"
		},
		{
			"Actions": [
				{
					"Action": "print",
					"Name": "catch",
					"Repeat": 1,
					"Request": {
						"message": "error"
					},
					"Service": "workflow",
					"Tag": "catch",
					"TagID": "catch"
				}
			],
			"Name": "catch"
		}
	]
}`,
		},
		{
			Description: "flat pipeline",
			YAMLData: `
pipeline:
  checkout:
    :action: vc:checkout
    origin:
      URL: http://github.com/adrianwit/echo
  build:
    :workflow: docker/build:build
    commands:
      - apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev
      - go get
      - go version
      - go build -a

`,
			Expected: `{
	"tasks": [
		{
			"Actions": [
				{
					"Action": "checkout",
					"Name": "checkout",
					"Repeat": 1,
					"Request": {
						"origin": [
							{
								"Key": "URL",
								"Value": "http://github.com/adrianwit/echo"
							}
						]
					},
					"Service": "vc",
					"Tag": "checkout",
					"TagID": "checkout"
				}
			],
			"Name": "checkout"
		},
		{
			"Actions": [
				{
					"Action": "run",
					"Name": "build",
					"Repeat": 1,
					"Request": {
						"URL": "docker/build.csv",
						"params": {
							"commands": [
								"apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev",
								"go get",
								"go version",
								"go build -a"
							]
						},
						"tasks": "build"
					},
					"Service": "workflow",
					"Tag": "build",
					"TagID": "build"
				}
			],
			"Name": "build"
		}
	]
}`,
		},

		{
			Description: "init/post pipeline",
			YAMLData: `
init:
  - "var1 = $var2"
defaults:
  d: v
post:
  - "var3 = $var10"
pipeline:
  checkout:
    action: "vc:checkout"
`,
			Expected: `{
	"Init": [
		{
			"Name": "var1",
			"Value": "$var2"
		}
	],
	"Post": [
		{
			"Name": "var3",
			"Value": "$var10"
		}
	],
	"tasks": [
		{
			"Actions": [
				{
					"Action": "checkout",
					"Name": "checkout",
					"Repeat": 1,
					"Request": {
						"d": "v"
					},
					"Service": "vc",
					"Tag": "checkout"
				}
			],
			"Name": "checkout"
		}
	]
}`,
		},

		{
			Description: "task with subtask",
			YAMLData: `pipeline:
  service:
    mysql:
      action: docker:run
    aerospike:
      action: docker:run
  catch:
    action: print
    message: error
  defer:
    action: print
    message: done`,
			Expected: `{
	"DeferredTask": "defer",
	"OnErrorTask": "catch",
	"tasks": [
		{
			"Name": "service",
			"tasks": [
				{
					"Actions": [
						{
							"Action": "run",
							"Name": "mysql",
							"Repeat": 1,
							"Service": "docker",
							"Tag": "mysql",
							"TagID": "_service"
						}
					],
					"Name": "mysql"
				},
				{
					"Actions": [
						{
							"Action": "run",
							"Name": "aerospike",
							"Repeat": 1,
							"Service": "docker",
							"Tag": "aerospike",
							"TagID": "_service"
						}
					],
					"Name": "aerospike"
				}
			]
		},
		{
			"Actions": [
				{
					"Action": "print",
					"Name": "catch",
					"Repeat": 1,
					"Request": {
						"message": "error"
					},
					"Service": "workflow",
					"Tag": "catch",
					"TagID": "catch"
				}
			],
			"Name": "catch"
		},
		{
			"Actions": [
				{
					"Action": "print",
					"Name": "defer",
					"Repeat": 1,
					"Request": {
						"message": "done"
					},
					"Service": "workflow",
					"Tag": "defer",
					"TagID": "defer"
				}
			],
			"Name": "defer"
		}
	]
}`,
		},

		{
			Description: "task with subtask, onError deferredTask",
			YAMLData: `pipeline:
  mysql:
    action: docker:run
  aerospike:
    action: docker:run
  catch:
    action: print
    message: error
  defer:
    action: print
    message: done`,
			Expected: `{
	"DeferredTask": "defer",
	"OnErrorTask": "catch",
	"tasks": [
		{
			"Actions": [
				{
					"Action": "run",
					"Name": "mysql",
					"Repeat": 1,
					"Service": "docker",
					"Tag": "mysql"
				}
			],
			"Name": "mysql"
		},
		{
			"Actions": [
				{
					"Action": "run",
					"Name": "aerospike",
					"Repeat": 1,
					"Service": "docker",
					"Tag": "aerospike"
				}
			],
			"Name": "aerospike"
		},
		{
			"Actions": [
				{
					"Action": "print",
					"Name": "catch",
					"Repeat": 1,
					"Request": {
						"message": "error"
					},
					"Service": "workflow",
					"Tag": "catch"
				}
			],
			"Name": "catch"
		},
		{
			"Actions": [
				{
					"Action": "print",
					"Name": "defer",
					"Repeat": 1,
					"Request": {
						"message": "done"
					},
					"Service": "workflow",
					"Tag": "defer"
				}
			],
			"Name": "defer"
		}
	]
}`,
		},

		{
			Description: "task with subtask init post",
			YAMLData: `pipeline:
  mysql:
    action: docker:run
    init:
      - key1 = $a
  aero:
    action: docker:run
    post:
      - key2 = $b

`,
			Expected: `{
	"tasks": [
		{
			"Actions": [
				{
					"Action": "run",
					"Init": [
						{
							"Name": "key1",
							"Value": "$a"
						}
					],
					"Name": "mysql",
					"Repeat": 1,
					"Service": "docker",
					"Tag": "mysql",
					"TagID": "mysql"
				}
			],
			"Name": "mysql"
		},
		{
			"Actions": [
				{
					"Action": "run",
					"Name": "aero",
					"Post": [
						{
							"Name": "key2",
							"Value": "$b"
						}
					],
					"Repeat": 1,
					"Service": "docker",
					"Tag": "aero",
					"TagID": "aero"
				}
			],
			"Name": "aero"
		}
	]
}`,
		},
	}

	for i, useCase := range useCases {
		inline := &Inlined{}
		var YAML = &yaml.MapSlice{}
		pipeline := map[string]interface{}{}
		err := yaml.NewDecoder(strings.NewReader(useCase.YAMLData)).Decode(YAML)
		if !assert.Nil(t, err, useCase.Description) {
			return
		}
		for _, entry := range *YAML {
			pipeline[toolbox.AsString(entry.Key)] = entry.Value
		}
		err = toolbox.DefaultConverter.AssignConverted(inline, pipeline)
		if !assert.Nil(t, err, useCase.Description) {
			return
		}
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		workflow, err := inline.AsWorkflow("", "")
		if !assert.Nil(t, err, useCase.Description) {
			continue
		}

		if !assertly.AssertValues(t, useCase.Expected, workflow, useCase.Description+fmt.Sprintf("%d", i)) {
			toolbox.DumpIndent(workflow, true)
		}
	}

}
