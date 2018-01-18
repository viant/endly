package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestDockerTag_Validate(t *testing.T) {

	{
		request := &endly.DockerTagRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerTagRequest{
			Target: url.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerTagRequest{
			Target:    url.NewResource("abc"),
			SourceTag: &endly.DockerTag{},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerTagRequest{
			Target:    url.NewResource("abc"),
			SourceTag: &endly.DockerTag{},
			TargetTag: &endly.DockerTag{},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.DockerTagRequest{
			Target:    url.NewResource("abc"),
			SourceTag: &endly.DockerTag{},
			TargetTag: &endly.DockerTag{
				Image: "abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{
				Image: "abc",
			},
			TargetTag: &endly.DockerTag{},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{
				Image: "abc",
			},
			TargetTag: &endly.DockerTag{
				Image: "abc",
			},
		}
		assert.Nil(t, request.Validate())
	}

}

func TestDockerTag_String(t *testing.T) {
	{
		tag := &endly.DockerTag{
			Image: "abc",
		}
		assert.EqualValues(t, "abc", tag.String())
	}
	{
		tag := &endly.DockerTag{
			Image:   "abc",
			Version: "latest",
		}
		assert.EqualValues(t, "abc:latest", tag.String())
	}
	{
		tag := &endly.DockerTag{
			Registry: "reg.org",
			Image:    "abc",
			Version:  "latest",
		}
		assert.EqualValues(t, "reg.org/abc:latest", tag.String())
	}
	{
		tag := &endly.DockerTag{
			Username: "reg.org",
			Image:    "abc",
			Version:  "latest",
		}
		assert.EqualValues(t, "reg.org/abc:latest", tag.String())
	}
}
