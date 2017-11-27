package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
	"path"
	"os"
)

func TestDaemonService_Status(t *testing.T) {

	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		service  string
		expected bool
	}{
		{
			"test/daemon/status/active/linux",
			target,
			"docker",
			true,
		},
		{
			"test/daemon/status/active/darwin",
			target,
			"docker",
			true,
		},
		{
			"test/daemon/status/inactive/linux",
			target,
			"mysql",
			false,
		},
		{
			"test/daemon/status/inactive/darwin",
			target,
			"mysql",
			false,
		},
		{
			"test/daemon/status/unknown/linux",
			target,
			"myabc",
			false,
		},
		{
			"test/daemon/status/unknown/darwin",
			target,
			"myabc",
			false,
		},
	}



	for _, useCase := range useCases {
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, useCase.target, execService)
			service, err := context.Service(endly.DaemonServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				response := service.Run(context, &endly.DaemonStatusRequest{
					Target:  target,
					Service: useCase.service,
				})
				assert.Equal(t, "", response.Error)
				info, ok := response.Response.(*endly.DaemonInfo)
				if assert.True(t, ok) && info != nil {
					assert.Equal(t, useCase.expected, info.IsActive(), useCase.baseDir + " " +  useCase.service)
				}
			}

		}

	}
}




func TestDaemonService_Run(t *testing.T) {

	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")

	//	var target = url.NewResource("scp://104.198.99.186:22/", credentialFile) //
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	manager := endly.NewManager()

	context, err := OpenTestRecorderContext(manager, target, "test/daemon/status/unknown/darwin")
	///context := manager.NewContext(toolbox.NewContext())

	defer context.Close()

	systemService, err := context.Service(endly.DaemonServiceID)
	assert.Nil(t, err)

	response := systemService.Run(context, &endly.DaemonStatusRequest{
		Target:  target,
		Service: "myabc",
	})

	assert.Equal(t, "", response.Error)
	info, ok := response.Response.(*endly.DaemonInfo)
	if assert.True(t, ok) && info != nil {
		assert.False(t, info.IsActive())
	}

}
