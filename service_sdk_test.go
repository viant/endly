package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestSdkService_Run(t *testing.T) {

	manager := endly.NewManager()
	service, err := manager.Service(endly.SdkServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	//context := manager.NewContext(toolbox.NewContext())
	//response := service.Run(context, &endly.SystemSdkSetRequest{
	//	Target: &url.Resource{
	//		URL: "scp://127.0.0.1/",
	//	},
	//	Sdk:     "jdk",
	//	Version: "1.7",
	//})
	//if response.Error != "" {
	//	sdkSetResponse, ok := response.Response.(*endly.SystemSdkSetResponse)
	//	assert.True(t, ok)
	//	assert.True(t, len(sdkSetResponse.SdkInfo.Build) > 0)
	//	assert.True(t, len(sdkSetResponse.SdkInfo.Home) > 0)
	//}
}
