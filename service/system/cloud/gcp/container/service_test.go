package container

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
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
