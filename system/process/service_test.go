package process_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/process"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestProcessService_Status(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	assert.Nil(t, err)
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.New()
	var useCases = []struct {
		baseDir string
		target  *url.Resource
		Command string
		Exected []*process.Info
	}{
		{
			"test/status/active/darwin",
			target,
			"docker",
			[]*process.Info{
				{
					Name:      "/Library/PrivilegedHelperTools/com.docker.vmnetd",
					Pid:       34227,
					Command:   "docker",
					Arguments: []string{},
					Stdin:     "ps -ef | grep docker",
					Stdout:    "0 34227 1 0 3:41PM ?? 0:00.01 /Library/PrivilegedHelperTools/com.docker.vmnetd",
				},
			},
		},
		{
			"test/status/active/linux",
			target,
			"docker",
			[]*process.Info{
				{
					Name:      "/usr/bin/dockerd",
					Pid:       1700,
					Command:   "docker",
					Arguments: []string{"-H", "fd://"},
					Stdin:     "ps -ef | grep docker",
					Stdout:    "root 1700 1 0 14:31 ? 00:00:29 /usr/bin/dockerd -H fd://",
				},
				{
					Name:      "docker-containerd",
					Pid:       1763,
					Command:   "docker",
					Arguments: []string{"-l", "unix:///var/run/docker/libcontainerd/docker-containerd.sock", "--metrics-interval=0", "--start-timeout", "2m", "--state-dir", "/var/run/docker/libcontainerd/containerd", "--shim", "docker-containerd-shim", "--runtime", "docker-runc"},
					Stdin:     "ps -ef | grep docker",
					Stdout:    "root 1763 1700 0 14:31 ? 00:00:18 docker-containerd -l unix:///var/run/docker/libcontainerd/docker-containerd.sock --metrics-interval=0 --start-timeout 2m --state-dir /var/run/docker/libcontainerd/containerd --shim docker-containerd-shim --runtime docker-runc",
				},
			},
		},
		{
			"test/status/inactive/darwin",
			target,
			"myabc",
			[]*process.Info{},
		},
		{
			"test/status/inactive/linux",
			target,
			"myabc",
			[]*process.Info{},
		},
	}

	for _, useCase := range useCases {
		context, err := exec.NewSSHReplayContext(manager, useCase.target, useCase.baseDir)
		if assert.Nil(t, err) {
			var request = &process.StatusRequest{
				Target:  useCase.target,
				Command: useCase.Command,
			}
			var response = &process.StatusResponse{}
			var description = useCase.baseDir + " " + useCase.Command
			err := endly.Run(context, request, response)
			if !assert.Nil(t, err, description) {
				continue
			}

			if len(response.Processes) != len(useCase.Exected) {
				assert.Fail(t, fmt.Sprintf("Expected %v processes info but had %v", len(useCase.Exected), len(response.Processes)))
			}

			for i, expected := range useCase.Exected {

				if i >= len(response.Processes) {
					assert.Fail(t, fmt.Sprintf("Process was missing [%v] %v", i, description))
					continue
				}
				var actual = response.Processes[i]
				assert.Equal(t, expected.Name, actual.Name, "name "+description)
				assert.Equal(t, expected.Command, actual.Command, "command "+description)
				assert.Equal(t, expected.Pid, actual.Pid, "pid "+description)
				assert.EqualValues(t, expected.Arguments, actual.Arguments, "command "+description)
				assert.Equal(t, expected.Stdin, actual.Stdin, "Stdin "+description)
				assert.Equal(t, expected.Stdout, actual.Stdout, "Stdout "+description)

			}

		}

	}
}
