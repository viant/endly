package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func TestSystemService_Run(t *testing.T) {

	var credentialFile = path.Join(os.Getenv("HOME"), "secret/scp.json")

	if toolbox.FileExists("/Library/LaunchDaemons/com.docker.vmnetd.plist") && toolbox.FileExists(credentialFile) {
		manager := endly.GetManager()
		context := manager.NewContext(toolbox.NewContext())
		defer context.Close()
		systemService, err := context.Service(endly.SystemServiceId)
		assert.Nil(t, err)

		{
			response := systemService.Run(context, &endly.ServiceStartRequest{
				Target: &endly.Resource{
					URL:            "scp://127.0.0.1/",
					CredentialFile: credentialFile,
				},
				Service: "docker",
			})

			assert.Equal(t, "", response.Error)
			info, ok := response.Response.(*endly.ServiceInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())
		}
		{
			response := systemService.Run(context, &endly.ServiceStatusRequest{
				Target: &endly.Resource{
					URL:            "scp://127.0.0.1/",
					CredentialFile: credentialFile,
				},
				Service: "docker",
			})

			assert.Equal(t, "", response.Error)
			info, ok := response.Response.(*endly.ServiceInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())
		}

		//{
		//	response := systemService.Run(context, &endly.ServiceStartRequest{
		//		Target: &endly.Resource{
		//			URL:            "scp://127.0.0.1/",
		//			CredentialFile: "/Users/awitas/secret/scp.json",
		//		},
		//		Service: "mysql",
		//	})
		//
		//	assert.Equal(t, "", response.Error)
		//	info, ok := response.ServiceResponse.(*endly.ServiceInfo)
		//	assert.True(t, ok)
		//	assert.True(t, info.IsActive())
		//
		//}
		//{
		//	response := systemService.Run(context, &endly.ServiceStatusRequest{
		//		Target: &endly.Resource{
		//			URL:            "scp://127.0.0.1/",
		//			CredentialFile: credentialFile,
		//		},
		//		Service: "mysql",
		//	})
		//
		//	assert.Equal(t, "", response.Error)
		//	info, ok := response.ServiceResponse.(*endly.ServiceInfo)
		//	assert.True(t, ok)
		//	assert.True(t, info.IsActive())
		//}

	}

}
