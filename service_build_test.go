package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestBuildService_Run(t *testing.T) {

	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	url := fmt.Sprintf("file://%v/build/meta/", parent)

	service := endly.GetBuildService()
	err := service.Load(&endly.BuildConfig{
		URL: []string{url},
	})
	if !assert.Nil(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, service)
	manager := endly.GetManager()
	manager.Register(service)

	context := manager.NewContext(toolbox.NewContext())
	assert.NotNil(t, context)

	buildService, err := manager.Service(endly.BuildServiceId)
	assert.Nil(t, err)

	response := buildService.Run(context, &endly.BuildRequest{
		BuildSpec: &endly.BuildSpec{
			Name: "maven",
			Goal: "package",
			Args: "-Dmvn.test.skip",
		},
		Target: &endly.Resource{
			URL: fmt.Sprintf("scp://127.0.0.1/%v/test/build/project1", parent),
		},
	})
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "", response.Error)

}
