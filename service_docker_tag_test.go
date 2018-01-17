package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox/url"
)

func TestDockerTag_Validate(t *testing.T) {

	{
		request := &endly.DockerServiceTagRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerServiceTagRequest{
			Target: url.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerServiceTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{

			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerServiceTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{

			},
			TargetTag: &endly.DockerTag{

			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.DockerServiceTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{

			},
			TargetTag: &endly.DockerTag{
				Image: "abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerServiceTagRequest{
			Target: url.NewResource("abc"),
			SourceTag: &endly.DockerTag{
				Image: "abc",
			},
			TargetTag: &endly.DockerTag{

			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &endly.DockerServiceTagRequest{
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
			Image: "abc",
			Version:"latest",
		}
		assert.EqualValues(t, "abc:latest", tag.String())
	}
	{
		tag := &endly.DockerTag{
			Registry:"reg.org",
			Image: "abc",
			Version:"latest",
		}
		assert.EqualValues(t, "reg.org/abc:latest", tag.String())
	}
	{
		tag := &endly.DockerTag{
			Username:"reg.org",
			Image: "abc",
			Version:"latest",
		}
		assert.EqualValues(t, "reg.org/abc:latest", tag.String())
	}
}
