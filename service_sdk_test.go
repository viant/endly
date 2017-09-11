package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestSdkService_Run(t *testing.T) {

	manager := endly.GetManager()
	service, err := manager.Service(endly.JsdServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())

	response := service.Run(context, &endly.SetSdkRequest{
		Target: &endly.Resource{
			URL: "scp://127.0.0.1/",
		},
		Sdk:     "jdk",
		Version: "1.7",
	})
	if response.Error != "" {
		info, ok := response.Response.(*endly.SetSdkResponse)
		assert.True(t, ok)
		assert.True(t, len(info.Build) > 0)
		assert.True(t, len(info.Home) > 0)
	}
}
