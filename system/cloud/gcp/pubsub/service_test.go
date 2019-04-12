package pubsub

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"testing"
)

func TestNew(t *testing.T) {

	context := endly.New().NewContext(nil)
	if !gcp.HasTestCredentials() {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)
	cred, _ := context.Secrets.GetCredentials("gcp-e2e")

	request, err := context.NewRequest(ServiceID, "subscriptionsList", map[string]interface{}{
		"project": fmt.Sprintf("projects/%v", cred.ProjectID),
	})
	assert.Nil(t, err)
	assert.NotNil(t, request)

}
