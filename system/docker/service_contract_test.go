package docker

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestNewRequestFromURL(t *testing.T) {

	var useCases = []struct {
		Description string
		URL         string
		Provider    func(URL string) interface{}
		Expected    interface{}
	}{
		{
			Description: "ImagesRequest",
			URL:         "test/req/images.yaml",
			Provider: func(URL string) interface{} {
				req, _ := NewImagesRequestFromURL(URL)
				return req
			},
			Expected: `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "localhost"
  },
  "Repository": "mysql",
  "Tag": "5.6"
}`,
		},
		{
			Description: "RunRequest JSON",
			URL:         "test/req/run.json",
			Provider: func(URL string) interface{} {
				req, _ := NewRunRequestFromURL(URL)
				return req
			},

			Expected: `{
  "Target": {
    "URL": "scp://127.0.0.1:22/",
    "Credentials": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/dummy20.json"
  },
  "Name": "testMysql",
  "Secrets": {
    "**mysql**": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/mysql20.json"
  },
  "Image": "mysql:5.6",
  "Port": "",
  "Env": {
    "MYSQL_ROOT_PASSWORD": "**mysql**"
  },
  "Mount": {
    "/tmp/my.cnf": "/etc/my.cnf"
  },
  "Ports": {
    "3306": "3306"
  }
}
`,
		},
		{
			Description: "RunRequest YAML",
			URL:         "test/req/run.yaml",
			Provider: func(URL string) interface{} {
				req, _ := NewRunRequestFromURL(URL)
				return req
			},

			Expected: `{
  "Target": {
    "URL": "scp://127.0.0.1:22/",
    "Credentials": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/dummy20.json"
  },
  "Name": "testMysql",
  "Secrets": {
    "**mysql**": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/mysql20.json"
  },
  "Image": "mysql:5.6",
  "Port": "",
  "Env": {
    "MYSQL_ROOT_PASSWORD": "**mysql**"
  },
  "Mount": {
    "/tmp/my.cnf": "/etc/my.cnf"
  },
  "Ports": {
    "3306": "3306"
  }
}
`,
		},

		{
			Description: "ExecRequest YAML",
			URL:         "test/req/exec.yaml",
			Provider: func(URL string) interface{} {
				req, _ := NewExecRequestFromURL(URL)
				return req
			},
			Expected: `{
  "Target": {
    "URL": "scp://127.0.0.1:22/",
    "Credentials": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/dummy20.json"
  },
  "Name": "testMysql",
  "Command": "mysqldump  -uroot -p***mysql*** --all-databases --routines | grep -v 'Warning' \u003e /tmp/dump.sql",
  "Secrets": {
    "***mysql***": "/var/folders/gl/5550g3kj6tn1rbz8chqx1c61ycmmm1/T/mysql20.json"
  },
  "Interactive": true,
  "AllocateTerminal": true,
  "RunInTheBackground": false
}
`,
		},
	}

	for _, useCase := range useCases {
		request := useCase.Provider(useCase.URL)
		assertly.AssertValues(t, useCase.Expected, request, useCase.Description)
	}
}

func TestContainerBaseRequest_Validate(t *testing.T) {
	{
		var stopRequest = &StopRequest{BaseRequest: &BaseRequest{}}
		assert.NotNil(t, stopRequest.Validate())
	}
	{
		var stopRequest = &StopRequest{BaseRequest: &BaseRequest{Target: url.NewResource("abc")}}
		assert.NotNil(t, stopRequest.Validate())
	}
}

func TestRunRequest_Validate(t *testing.T) {
	{
		var stopRequest = &RunRequest{}
		assert.NotNil(t, stopRequest.Validate())
	}
	{
		var stopRequest = &RunRequest{Target: url.NewResource("abc")}
		assert.NotNil(t, stopRequest.Validate())
	}
}
