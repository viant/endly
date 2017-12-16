package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox/url"
)

func Test_MatchVersion(t *testing.T) {
	assert.True(t, endly.MatchVersion("10.2", "10.2.1"))
	assert.True(t, endly.MatchVersion("10.2.1", "10.2"))

	assert.False(t, endly.MatchVersion("10.1", "10.2.1"))

	assert.True(t, endly.MatchVersion("10.2.1", "10.2.1"))

}


func Test_DeplymentValiate(t *testing.T) {

	{
		deployment := &endly.Deployment{}

		err := deployment.Validate()
		assert.NotNil(t, err)
	}

	{
		deployment := &endly.Deployment{
			Transfer:&endly.Transfer{},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}

	{
		deployment := &endly.Deployment{
			Transfer:&endly.Transfer{
				Target:&url.Resource{},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}
	{
		deployment := &endly.Deployment{
			Transfer:&endly.Transfer{
				Target:&url.Resource{URL:"mem:///123"},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}
	{
		deployment := &endly.Deployment{
			Transfer:&endly.Transfer{
				Target:&url.Resource{URL:"mem:///123"},
				Source:&url.Resource{},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}

}