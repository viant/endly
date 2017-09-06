package sdk_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/viant/endly/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
)

func TestSdkService_Run(t *testing.T) {

	manager := endly.NewManager()
	service, err := manager.Service(sdk.JsdServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)


	context := manager.NewContext(toolbox.NewContext())

	response := service.Run(context, &sdk.SetSdkRequest{
		Target:&endly.Resource{
			URL:"scp://127.0.0.1/",
		},
		Sdk:"jdk",
		Version:"1.7",
	})
	if response.Error != nil {
		info, ok := response.Response.(*sdk.SetSdkResponse)
		assert.True(t, ok)
		assert.True(t, len(info.Build) > 0)
		assert.True(t, len(info.Home) > 0)
	}
}