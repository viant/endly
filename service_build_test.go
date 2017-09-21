package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestBuildService_Run(t *testing.T) {

	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	assert.NotNil(t, context)

	buildService, err := manager.Service(endly.BuildServiceId)
	assert.Nil(t, err)

	response := buildService.Run(context, &endly.BuildRequest{
		BuildSpec: &endly.BuildSpec{
			Name:       "maven",
			Goal:       "package",
			Args:       "-Dmvn.test.skip",
			Sdk:        "jdk",
			SdkVersion: "1.7",
		},
		Target: endly.NewFileResource("test/build/project1"),
	})
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "", response.Error)

}
