package gce_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/cloud/gce"
	"testing"
)

func TestGCECallRequest_Validate(t *testing.T) {

	{
		request := gce.CallRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := gce.CallRequest{Service: "Instances"}
		assert.NotNil(t, request.Validate())
	}
	{
		request := gce.CallRequest{Service: "Instances", Credential: "abc"}
		assert.NotNil(t, request.Validate())
	}
	{
		request := gce.CallRequest{Method: "Get", Credential: "abc"}
		assert.NotNil(t, request.Validate())
	}

	{
		request := gce.CallRequest{Service: "Instances", Credential: "abc", Method: "Get"}
		assert.Nil(t, request.Validate())
	}
}
