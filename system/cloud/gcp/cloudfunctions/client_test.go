package cloudfunctions

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
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
}
