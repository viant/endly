package exec_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestNewExecService(t *testing.T) {

	var useCases = []struct {
		description string
		baseDir     string
		target      *url.Resource
		expected    *endly.OperatingSystem
	}{
		{
			description: "open new session on linux",
			baseDir:     "test/open/linux",
			target:      url.NewResource("ssh://127.0.0.1:22/etc"),
			expected: &endly.OperatingSystem{Name: "ubuntu", Architecture: "x64", Hardware: "x86_64", Version: "17.04", System: "linux", Path: &endly.SystemPath{
				Items: []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin", "/usr/games", "/usr/local/games"},
			}},
		},
		{
			description: "open new session on osx",
			baseDir:     "test/open/darwin",
			target:      url.NewResource("ssh://127.0.0.1:22/etc"),
			expected: &endly.OperatingSystem{Name: "macosx", Architecture: "x64", Hardware: "x86_64", Version: "10.12.6", System: "darwin", Path: &endly.SystemPath{
				Items: []string{"/usr/local/apache-maven-3.2.5/bin", "/usr/local/opt/libpcap/bin", "/usr/libexec/", "/Projects/go/workspace/bin", "/usr/local/apache-maven-3.2.5/bin", "/usr/local/opt/libpcap/bin", "/usr/libexec/", "/Projects/go/workspace/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"},
			}},
		},
	}

	manager := endly.New()
	for _, useCase := range useCases {
		service, err := exec.GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := exec.OpenTestContext(manager, useCase.target, service)
			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				actual := context.OperatingSystem(target.Host())
				if assert.NotNil(t, actual) {
					expected := useCase.expected
					assert.Equal(t, expected.Name, actual.Name, "os.name")
					assert.Equal(t, expected.Version, actual.Version, "os.version")
					assert.Equal(t, expected.Hardware, actual.Hardware, "os.hardware")
					assert.Equal(t, expected.System, actual.System, "os.system")
					assert.EqualValues(t, expected.Path.Items, actual.Path.Items, "os.path")
				}
			}

		}

	}

}

func Test_NewSimpleCommandRequest(t *testing.T) {
	command := exec.NewSimpleRunRequest(url.NewResource("scp://127.0.0.1"), "ls -al")
	assert.EqualValues(t, "ls -al", command.ExtractableCommand.Executions[0].Command)
}

// Function template  to capture SSH conversation
//func TestXXXXService_Run(t *testing.T) {
//
//	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/localhost.json")
//
//	//var target = url.NewResource("scp://35.197.115.53:22/", credentialFile) //
//	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
//	manager := endly.New()
//
//	context, err := OpenTestRecorderContext(manager, target, "test/daemon/start/unknown/darwin")
//	///context := manager.NewContext(toolbox.NewContext())
//
//	defer context.Close()
//
//	systemService, err := context.Service(endly.XXXServiceID)
//	assert.Nil(t, err)
//
//	response := systemService.Run(context, &endly.XXXStartRequest{
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
