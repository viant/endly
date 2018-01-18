package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestDockerLoginRequest_Validate(t *testing.T) {
	{
		request := &endly.DockerLoginRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerLoginRequest{
			Target: url.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerLoginRequest{
			Repository: "abc",
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerLoginRequest{
			Repository: "abc",
			Target:     url.NewResource("abc"),
		}
		assert.Nil(t, request.Validate())
	}
}
