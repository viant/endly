package run

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"log"
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
		Image:   "gcr.io/cloudrun/hello",
		Replace: true,
		Public:  true,
	}
	deployResponse := &DeployResponse{}
	err = endly.Run(context, request, &deployResponse)
	if !assert.Nil(t, err) {
		log.Print(err)
	}
}
