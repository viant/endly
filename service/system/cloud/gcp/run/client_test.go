package run

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
	"google.golang.org/api/run/v1alpha1"
	"testing"
)

func TestClient(t *testing.T) {
	context := endly.New().NewContext(nil)
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "4234234dasdasde",
	})
	assert.NotNil(t, err)
	_, err = GetClient(context)
	assert.NotNil(t, err)
	if !gcp.HasTestCredentials() {
		return
	}
	err = InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)
	client, err := GetClient(context)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	service, ok := client.Service().(*run.Service)
	if !assert.True(t, ok) {
		return
	}
	assert.NotNil(t, service)
}
