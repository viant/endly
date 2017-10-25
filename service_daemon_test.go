package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"os"
	"path"
	"testing"
)

func TestDaemonService_Run(t *testing.T) {

	var credentialFile = path.Join(os.Getenv("HOME"), "secret/scp.json")

	if toolbox.FileExists("/Library/LaunchDaemons/com.docker.vmnetd.plist") && toolbox.FileExists(credentialFile) {
		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		defer context.Close()
		systemService, err := context.Service(endly.DaemonServiceID)
		assert.Nil(t, err)

		{
			response := systemService.Run(context, &endly.DaemonStartRequest{
				Target: &url.Resource{
					URL:        "scp://127.0.0.1/",
					Credential: credentialFile,
				},
				Service: "mysql",
			})

			assert.Equal(t, "", response.Error)
			info, ok := response.Response.(*endly.DaemonInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())
		}
		{
			response := systemService.Run(context, &endly.DaemonStatusRequest{
				Target: &url.Resource{
					URL:        "scp://127.0.0.1/",
					Credential: credentialFile,
				},
				Service: "mysql",
			})

			assert.Equal(t, "", response.Error)
			info, ok := response.Response.(*endly.DaemonInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())
		}

		//{
		//	response := systemService.Run(context, &endly.DaemonStartRequest{
		//		Target: &url.Resource{
		//			URL:            "scp://127.0.0.1/",
		//			 Credential: "/Users/awitas/secret/scp.json",
		//		},
		//		Service: "mysql",
		//	})
		//
		//	assert.Equal(t, "", response.Error)
		//	info, ok := response.ServiceResponse.(*endly.DaemonInfo)
		//	assert.True(t, ok)
		//	assert.True(t, info.IsActive())
		//
		//}
		//{
		//	response := systemService.Run(context, &endly.DaemonStatusRequest{
		//		Target: &url.Resource{
		//			URL:            "scp://127.0.0.1/",
		//			 Credential: credentialFile,
		//		},
		//		Service: "mysql",
		//	})
		//
		//	assert.Equal(t, "", response.Error)
		//	info, ok := response.ServiceResponse.(*endly.DaemonInfo)
		//	assert.True(t, ok)
		//	assert.True(t, info.IsActive())
		//}

	}

}
