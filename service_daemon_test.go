package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestDaemonService_Status(t *testing.T) {

	var credentialFile, err = GetDummyCredentail()
	assert.Nil(t, err)
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		service  string
		expected bool
		pid int
	}{
		{
			"test/daemon/status/active/darwin",
			target,
			"docker",
			true,
			35323,
		},
		{
			"test/daemon/status/active/linux",
			target,
			"docker",
			true,
			1609,
		},

		{
			"test/daemon/status/inactive/linux",
			target,
			"mysql",
			false,
			16436,
		},
		{
			"test/daemon/status/inactive/darwin",
			target,
			"mysql",
			false,
			0,
		},
		{
			"test/daemon/status/unknown/linux",
			target,
			"myabc",
			false,
			0,
		},
		{
			"test/daemon/status/unknown/darwin",
			target,
			"myabc",
			false,
			0,
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
				var baseCase = useCase.baseDir + " " +  useCase.service
				assert.Equal(t, "", response.Error, baseCase)
				info, ok := response.Response.(*endly.DaemonInfo)
				if assert.True(t, ok) && info != nil {
					assert.Equal(t, useCase.expected, info.IsActive(), "is running " + baseCase)
					assert.Equal(t, useCase.pid, info.Pid, "pid :" +  baseCase)
				}
			}

		}

	}
}


func TestDaemonService_Start(t *testing.T) {

	var credentialFile, err = GetDummyCredentail()
	assert.Nil(t, err)
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		service  string
		expected bool
		pid int
	}{

		{
			"test/daemon/start/inactive/linux",
			target,
			"docker",
			true,
			14124,
		},
		{
			"test/daemon/start/inactive/darwin",
			target,
			"docker",
			true,
			34514,
		},
		{
			"test/daemon/start/active/darwin",
			target,
			"docker",
			true,
			35323,
		},
		{
			"test/daemon/start/active/linux",
			target,
			"docker",
			true,
			14124,
		},
		{
			"test/daemon/start/unknown/linux",
			target,
			"myabc",
			false,
			0,
		},
		{
			"test/daemon/start/unknown/darwin",
			target,
			"myabc",
			false,
			0,
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
				response := service.Run(context, &endly.DaemonStartRequest{
					Target:  target,
					Service: useCase.service,
				})
				var baseCase = useCase.baseDir + " " +  useCase.service
				assert.Equal(t, "", response.Error, baseCase)
				info, ok := response.Response.(*endly.DaemonInfo)
				if assert.True(t, ok) && info != nil {
					assert.Equal(t, useCase.expected, info.IsActive(), "is running " + baseCase)
					assert.Equal(t, useCase.pid, info.Pid, "pid :" +  baseCase)
				}
			}

		}

	}
}



func TestDaemonService_Stop(t *testing.T) {

	var credentialFile, err = GetDummyCredentail()
	assert.Nil(t, err)
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		service  string
		expected bool
		pid int
	}{

		{
			"test/daemon/stop/inactive/linux",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/daemon/stop/inactive/darwin",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/daemon/stop/active/darwin",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/daemon/stop/active/linux",
			target,
			"docker",
			false,
			23828,
		},
		{
			"test/daemon/stop/unknown/linux",
			target,
			"myabc",
			false,
			0,
		},
		{
			"test/daemon/stop/unknown/darwin",
			target,
			"myabc",
			false,
			0,
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
				response := service.Run(context, &endly.DaemonStopRequest{
					Target:  target,
					Service: useCase.service,
				})
				var baseCase = useCase.baseDir + " " +  useCase.service
				assert.Equal(t, "", response.Error, baseCase)
				info, ok := response.Response.(*endly.DaemonInfo)
				if assert.True(t, ok) && info != nil {
					assert.Equal(t, useCase.expected, info.IsActive(), "is running " + baseCase)
					assert.Equal(t, useCase.pid, info.Pid, "pid :" +  baseCase)
				}
			}

		}

	}
}


//
//func TestDaemonService_Run(t *testing.T) {
//
//	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")
//
//	//var target = url.NewResource("scp://35.197.115.53:22/", credentialFile) //
//	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
//	manager := endly.NewManager()
//
//	context, err := OpenTestRecorderContext(manager, target, "test/daemon/start/unknown/darwin")
//	///context := manager.NewContext(toolbox.NewContext())
//
//	defer context.Close()
//
//	systemService, err := context.Service(endly.DaemonServiceID)
//	assert.Nil(t, err)
//
//	response := systemService.Run(context, &endly.DaemonStartRequest{
//		Target:  target,
//		Service: "myabc",
//	})
//
//	assert.Equal(t, "", response.Error)
//	info, ok := response.Response.(*endly.DaemonInfo)
//	if assert.True(t, ok) && info != nil {
//		assert.False(t, info.IsActive())
//	}
//
//}
