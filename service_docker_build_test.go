package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func Test_DockerBuildRequest_Validate(t *testing.T) {
	{
		request := endly.DockerBuildRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := endly.DockerBuildRequest{Target: url.NewResource("abc"), Tag: &endly.DockerTag{}}
		assert.NotNil(t, request.Validate())
	}
	{
		request := endly.DockerBuildRequest{Target: url.NewResource("abc"),
			Arguments: map[string]string{
				"-t": "image:1.0",
			},
			Tag: &endly.DockerTag{Image: "abc"}}
		assert.Nil(t, request.Validate())
	}

	{
		request := endly.DockerBuildRequest{Target: url.NewResource("abc"),

			Tag: &endly.DockerTag{Image: "abc"}}
		assert.Nil(t, request.Validate())
	}
}
