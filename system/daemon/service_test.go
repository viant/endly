package daemon_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/system/daemon"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"testing"
)

func TestDaemonService_Status(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	assert.Nil(t, err)
	var target = location.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		target   *location.Resource
		service  string
		expected bool
		pid      int
	}{
		{
			"test/status/active/darwin",
			target,
			"docker",
			true,
			35323,
		},
		{
			"test/status/active/linux",
			target,
			"docker",
			true,
			1609,
		},

		{
			"test/status/inactive/linux",
			target,
			"mysql",
			false,
			16436,
		},
		{
			"test/status/inactive/darwin",
			target,
			"mysql",
			false,
			0,
		},
		{
			"test/status/unknown/linux",
			target,
			"myabc",
			false,
			0,
		},
		{
			"test/status/unknown/darwin",
			target,
			"myabc",
			false,
			0,
		},
	}

	for i, useCase := range useCases {
		context, err := exec.NewSSHReplayContext(manager, useCase.target, useCase.baseDir)
		defer context.Close()
		if assert.Nil(t, err) {
			var response = &daemon.Info{}
			err := endly.Run(context, &daemon.StatusRequest{
				Target:  useCase.target,
				Service: useCase.service,
			}, response)
			var baseCase = useCase.baseDir + " " + useCase.service + fmt.Sprintf("[%d]", i)
			if !assert.Nil(t, err, baseCase) {
				continue
			}

			assert.Equal(t, useCase.expected, response.IsActive(), "is running "+baseCase)
			assert.Equal(t, useCase.pid, response.Pid, "pid :"+baseCase)
		}
	}
}

func TestDaemonService_Start(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	assert.Nil(t, err)
	var target = location.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		target   *location.Resource
		service  string
		expected bool
		pid      int
		Error    string
	}{

		{
			"test/start/inactive/linux",
			target,
			"docker",
			true,
			14124,
			"",
		},
		{
			"test/start/inactive/darwin",
			target,
			"docker",
			true,
			34514,
			"",
		},
		{
			"test/start/active/darwin",
			target,
			"docker",
			true,
			35323,
			"",
		},
		{
			"test/start/active/linux",
			target,
			"docker",
			true,
			14124,
			"",
		},
		{
			"test/start/unknown/linux",
			target,
			"myabc",
			false,
			0,
			"myabc service is inactive at daemon.start",
		},
		{
			"test/start/unknown/darwin",
			target,
			"myabc",
			false,
			0,
			"myabc service is inactive at daemon.start",
		},
	}

	for _, useCase := range useCases {
		context, err := exec.NewSSHReplayContext(manager, useCase.target, useCase.baseDir)
		if assert.Nil(t, err) {
			service, err := context.Service(daemon.ServiceID)
			assert.Nil(t, err)
			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				response := service.Run(context, &daemon.StartRequest{
					Target:  target,
					Service: useCase.service,
				})
				var baseCase = useCase.baseDir + " " + useCase.service

				if !assert.Equal(t, useCase.Error, response.Error, baseCase) {
					continue
				}

				info, ok := response.Response.(*daemon.StartResponse)
				if assert.True(t, ok) && info != nil {
					assert.Equal(t, useCase.expected, info.IsActive(), "is running "+baseCase)
					assert.Equal(t, useCase.pid, info.Pid, "pid :"+baseCase)
				}
			}
		}
	}
}

func TestDaemonService_Stop(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	assert.Nil(t, err)
	var target = location.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		target   *location.Resource
		service  string
		expected bool
		pid      int
	}{

		{
			"test/stop/inactive/linux",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/stop/inactive/darwin",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/stop/active/darwin",
			target,
			"docker",
			false,
			0,
		},
		{
			"test/stop/active/linux",
			target,
			"docker",
			false,
			23828,
		},
		{
			"test/stop/unknown/linux",
			target,
			"myabc",
			false,
			0,
		},
		{
			"test/stop/unknown/darwin",
			target,
			"myabc",
			false,
			0,
		},
	}

	for _, useCase := range useCases {
		context, err := exec.NewSSHReplayContext(manager, useCase.target, useCase.baseDir)
		if assert.Nil(t, err) {
			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				var response = &daemon.StopResponse{}
				err := endly.Run(context, &daemon.StopRequest{
					Target:  target,
					Service: useCase.service,
				}, response)
				var baseCase = useCase.baseDir + " " + useCase.service

				if !assert.Nil(t, err) {
					continue
				}
				assert.Equal(t, useCase.expected, response.IsActive(), "is running "+baseCase)
				assert.Equal(t, useCase.pid, response.Pid, "pid :"+baseCase)
			}

		}

	}
}
