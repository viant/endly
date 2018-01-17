package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func TestGCECallRequest_Validate(t *testing.T) {

	{
		request := endly.GCECallRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := endly.GCECallRequest{Service: "Instances"}
		assert.NotNil(t, request.Validate())
	}
	{
		request := endly.GCECallRequest{Service: "Instances", Credential: "abc"}
		assert.NotNil(t, request.Validate())
	}
	{
		request := endly.GCECallRequest{Method: "Get", Credential: "abc"}
		assert.NotNil(t, request.Validate())
	}

	{
		request := endly.GCECallRequest{Service: "Instances", Credential: "abc", Method: "Get"}
		assert.Nil(t, request.Validate())
	}
}
