package deploy_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/deployment/deploy"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox/url"
	"testing"
)

func Test_MatchVersion(t *testing.T) {
	assert.True(t, deploy.MatchVersion("10.2", "10.2.1"))
	assert.True(t, deploy.MatchVersion("10.2.1", "10.2"))

	assert.False(t, deploy.MatchVersion("10.1", "10.2.1"))

	assert.True(t, deploy.MatchVersion("10.2.1", "10.2.1"))

}

func Test_DeplymentValiate(t *testing.T) {

	{
		deployment := &deploy.Deployment{}

		err := deployment.Validate()
		assert.NotNil(t, err)
	}

	{
		deployment := &deploy.Deployment{
			Transfer: &storage.Transfer{},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}

	{
		deployment := &deploy.Deployment{
			Transfer: &storage.Transfer{
				Target: &url.Resource{},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}
	{
		deployment := &deploy.Deployment{
			Transfer: &storage.Transfer{
				Target: &url.Resource{URL: "mem:///123"},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}
	{
		deployment := &deploy.Deployment{
			Transfer: &storage.Transfer{
				Target: &url.Resource{URL: "mem:///123"},
				Source: &url.Resource{},
			},
		}
		err := deployment.Validate()
		assert.NotNil(t, err)
	}

}
