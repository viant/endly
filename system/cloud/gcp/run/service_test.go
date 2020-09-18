package run

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"testing"
)

func TestNew(t *testing.T) {

	if !gcp.HasTestCredentials() {
		return
	}

	context := endly.New().NewContext(nil)
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)

}

func TestService_Deploy(t *testing.T) {
	if !gcp.HasTestCredentials() {
		return
	}
	context := endly.New().NewContext(nil)
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})

	assert.Nil(t, err)
	request := &DeployRequest{
		Image:   "us.gcr.io/viant-e2e/sitelistmatch:latest",
		Replace: true,
		Public:  true,
	}
	request.Init()
	deployResponse := &DeployResponse{}
	err = endly.Run(context, request, &deployResponse)
	if !assert.Nil(t, err) {
		fmt.Printf("%v\n", err)
	}
}
