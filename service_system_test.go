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

	if toolbox.FileExists("/Library/LaunchDaemons/com.oracle.oss.mysql.mysqld.plist") && toolbox.FileExists(credentialFile) {
		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		defer context.Close()
		systemService, err := context.Service(endly.SystemServiceId)
		assert.Nil(t, err)

		{
			response := systemService.Run(context, &endly.StartServiceRequest{
				Target: &endly.Resource{
					URL:            "scp://127.0.0.1/",
					CredentialFile: "/Users/awitas/secret/scp.json",
				},
				Service: "mysql",
			})

			assert.Nil(t, response.Error)
			info, ok := response.Response.(*endly.ServiceInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())

		}
		{
			response := systemService.Run(context, &endly.StatusServiceRequest{
				Target: &endly.Resource{
					URL:            "scp://127.0.0.1/",
					CredentialFile: credentialFile,
				},
				Service: "mysql",
			})

			assert.Nil(t, response.Error)
			info, ok := response.Response.(*endly.ServiceInfo)
			assert.True(t, ok)
			assert.True(t, info.IsActive())
		}
	}

}
