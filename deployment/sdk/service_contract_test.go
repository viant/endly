package sdk

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestSetRequest_Init(t *testing.T) {
	var req = NewSetRequest(url.NewResource("abc"), "go:1.8", "", nil)
	assert.Nil(t, req.Init())
	assert.EqualValues(t, "go", req.Sdk)
	assert.EqualValues(t, "1.8", req.Version)
	assert.Nil(t, req.Validate())
}

func TestSetRequest_Validate(t *testing.T) {

	{ //version
		var req = NewSetRequest(url.NewResource("abc"), "go:1.8", "", nil)
		assert.NotNil(t, req.Validate())
	}
	{ //sdk
		var req = NewSetRequest(url.NewResource("abc"), "", "1.3", nil)
		assert.NotNil(t, req.Validate())
	}
	{ //target
		var req = NewSetRequest(nil, "go:1.8", "", nil)
		assert.NotNil(t, req.Validate())
	}

}
