package compute

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
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
	request, err := context.NewRequest(ServiceID, "instancesList", map[string]interface{}{
		"zone": "us-central1-f",
	})
	assert.Nil(t, err)
	assert.NotNil(t, request)
}
