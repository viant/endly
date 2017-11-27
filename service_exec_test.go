package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestNewExecService(t *testing.T) {

	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		expected *endly.OperatingSystem
	}{
		{
			"test/exec/open/linux",
			url.NewResource("ssh://127.0.0.1:22/etc"),
			&endly.OperatingSystem{Name: "ubuntu", Architecture: "x64", Hardware: "x86_64", Version: "17.04", System:"linux", Path: &endly.SystemPath{
				SystemPath: []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin", "/usr/games", "/usr/local/games"},
			}},
		},
		{
			"test/exec/open/darwin",
			url.NewResource("ssh://127.0.0.1:22/etc"),
			&endly.OperatingSystem{Name: "macosx", Architecture: "x64", Hardware: "x86_64", Version: "10.12.6", System:"darwin", Path: &endly.SystemPath{
				SystemPath: []string{"/usr/local/apache-maven-3.2.5/bin", "/usr/local/opt/libpcap/bin", "/usr/libexec/", "/Projects/go/workspace/bin", "/usr/local/apache-maven-3.2.5/bin", "/usr/local/opt/libpcap/bin", "/usr/libexec/", "/Projects/go/workspace/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"},
			}},
		},
	}

	manager := endly.NewManager()
	for _, useCase := range useCases{
		service, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, useCase.target, service)
			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				actual := context.OperatingSystem(target.Host())
				if  assert.NotNil(t, actual) {
					expected := useCase.expected
					assert.Equal(t, expected.Name, actual.Name, "os.name")
					assert.Equal(t, expected.Version, actual.Version, "os.version")
					assert.Equal(t, expected.Hardware, actual.Hardware, "os.hardware")
					assert.Equal(t, expected.System, actual.System, "os.system")
					assert.EqualValues(t, expected.Path.SystemPath, actual.Path.SystemPath, "os.path")
				}
			}

		}

	}

}
