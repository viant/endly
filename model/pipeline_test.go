package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestInlinePipeline_Init(t *testing.T) {

	useCases := []struct {
		Description   string
		YAMLData      string
		Expected      interface{}
		DefaultParams map[string]interface{}
	}{
		{
			Description: "flat pipeline",
			YAMLData: `
pipeline:
  checkout:
    "@action": vc:checkout
    origin:
      URL: http://github.com/adrianwit/echo
  build:
    "@workflow": docker/build:build
    commands:
      - apt-get update; apt-get -y install libpcap0.8 libpcap0.8-dev
      - go get
      - go version
      - go build -a

`,
			Expected: `{"Pipelines":[
    {
      "Name": "checkout",
      "Action": "vc:checkout",
      "Params": {
        "origin": {
          "URL": "http://github.com/adrianwit/echo"
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
      ]
      }
    }
  ]}`,
		},
		{
			Description: "nested pipeline",
			YAMLData: `
pipeline:
  create-db:
    "@action": dsunit:register
    scripts:
      - URL: data/db1/schema.ddl
    datastore: db1
    recreate: true
    config:
      driverName: mysql
      descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
      credentials: $mysqlCredentials
    admin:
      datastore: mysql
      config:
        driverName: mysql
        descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
        credentials: $mysqlCredentials
  populate:
    "@action": dsunit:prepare
    datastore: db1
    URL: datastore/db1/dictionary
`,
			DefaultParams: map[string]interface{}{
				"key":       1,
				"datastore": "db1",
			},
			Expected: `{"Pipelines":[
  {
    "Name": "create-db",
    "Workflow": "",
    "Action": "dsunit:register",
    "Params": {
      "admin": {
        "config": {
          "credentials": "$mysqlCredentials",
          "descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
          "driverName": "mysql"
        },
        "datastore": "mysql"
      },
      "config": {
        "credentials": "$mysqlCredentials",
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
]}`,
		},

		{
			Description: "init/post pipeline",
			YAMLData: `
init: 
  - "var1 = $var2"
post: 
  - "var3 = $var10"
pipeline: 
  checkout: 
    action: "vc:checkout"
`,
			Expected: `{
  "Pipelines": [
    {
      "Name": "checkout",
      "Action": "vc:checkout",
      "Params": {
        "action": "vc:checkout"
      }
    }
  ],
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
  ]
}`,
		},
	}

	for _, useCase := range useCases {
		inline := &Inline{}
		var mapSlice = yaml.MapSlice{}
		err := yaml.NewDecoder(strings.NewReader(useCase.YAMLData)).Decode(&mapSlice)
		if !assert.Nil(t, err, useCase.Description) {
			return
		}
		err = toolbox.DefaultConverter.AssignConverted(inline, mapSlice)
		if !assert.Nil(t, err, useCase.Description) {
			return
		}
		err = inline.InitTasks("", TasksSelector(""), useCase.DefaultParams)
		if !assert.Nil(t, err, useCase.Description) {
			return
		}
		assertly.AssertValues(t, useCase.Expected, inline, useCase.Description)

	}

}
