package build_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/build"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestBuildService_Run(t *testing.T) {

	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	url := fmt.Sprintf("file://%v/meta/", parent)

	fmt.Printf("D: %v\n", url)

	service, err := build.NewBuildService(&build.Config{
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

	buildService, err := manager.Service(build.BuildServiceId)
	assert.Nil(t, err)

	response := buildService.Run(context, &build.BuildRequest{
		BuildSpec: &build.BuildSpec{
			Name: "maven",
			Goal: "package",
			Args: "-Dmvn.test.skip",
		},
		Target: &endly.Resource{
			URL: fmt.Sprintf("scp://127.0.0.1/%v/test/project1", parent),
		},
	})
	assert.Equal(t, "ok", response.Status)
	assert.Nil(t, response.Error)

}
