package docker

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestService_Build(t *testing.T) {
	service := New()
	assert.NotNil(t, service)

	parent := toolbox.CallerDirectory(3)

	response := &BuildResponse{}
	build := &BuildRequest{
		Tag: &Tag{
			Image:   "builder",
			Version: "1.0",
		},
		Path: path.Join(parent, "test/build/"),
	}
	assert.Nil(t, build.Init())
	err := endly.Run(nil, build, response)
	if !assert.Nil(t, err) {
		fmt.Printf("%v", err)
	}
	assert.True(t, response.ImageID != "")
}
