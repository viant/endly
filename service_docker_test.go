package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func getDockerService(manager endly.Manager) endly.Service {
	context := manager.NewContext(toolbox.NewContext())
	service := endly.NewDockerService()
	service.Run(context, &endly.DockerSystemPathRequest{
		SysPath: []string{"/usr/local/bin"},
	})
	return service
}

func TestNewDockerService(t *testing.T) {

	manager := endly.NewManager()

	context := manager.NewContext(toolbox.NewContext())
	service := getDockerService(manager)

	response := service.Run(context, &endly.DockerImagesRequest{
		Target: url.NewResource("ssh://127.0.0.1/"),
	})

	assert.Equal(t, "", response.Error)
	info, ok := response.Response.([]*endly.DockerImageInfo)
	assert.True(t, ok)
	assert.NotNil(t, info)
}

func TestNewDockerService_Pull(t *testing.T) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service := getDockerService(manager)

	response := service.Run(context, &endly.DockerPullRequest{
		Target: &url.Resource{
			URL: "ssh://127.0.0.1/",
		},
		Repository: "mysql",
		Tag:        "5.6",
	})
	assert.Equal(t, "", response.Error)
	info, ok := response.Response.(*endly.DockerImageInfo)
	assert.True(t, ok)
	assert.NotNil(t, info)
	assert.Equal(t, "mysql", info.Repository)
	assert.Equal(t, "5.6", info.Tag)
}

func TestNewDockerService_Run(t *testing.T) {

	credential := path.Join(os.Getenv("HOME"), "secret/mysql.json")
	if toolbox.FileExists(credential) {
		fileName, _, _ := toolbox.CallerInfo(2)
		parent, _ := path.Split(fileName)

		data, err := ioutil.ReadFile(path.Join(parent, "test/docker/my.cnf"))
		if err != nil {
			assert.FailNow(t, err.Error())
		}
		ioutil.WriteFile("/tmp/my.cnf", data, os.FileMode(0x644))

		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		service := getDockerService(manager)

		response := service.Run(context, &endly.DockerRunRequest{
			Target: &url.Resource{
				URL:  "ssh://127.0.0.1/",
				Name: "testmysql",
			},
			Image: "mysql:5.6",
			MappedPort: map[string]string{
				"3306": "3306",
			},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "****",
			},
			Mount: map[string]string{
				"/tmp/my.cnf": "/etc/my.cnf",
			},
			Credential: credential,
		})
		assert.Equal(t, "", response.Error)
		info, ok := response.Response.(*endly.DockerContainerInfo)
		assert.True(t, ok)
		if assert.NotNil(t, info) {
			assert.Equal(t, "mysql:5.6", info.Image)
			assert.Equal(t, "testmysql", info.Names)
		}

		response = service.Run(context, &endly.DockerContainerStopRequest{
			Target: &url.Resource{
				URL:  "ssh://127.0.0.1/",
				Name: "testmysql",
			},
		})
		assert.Equal(t, "", response.Error)

	}
}
