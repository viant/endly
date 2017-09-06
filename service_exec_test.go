package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewExecService(t *testing.T) {
	manager := endly.NewServiceManager()
	srv, err := manager.Service(endly.ExecServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, srv)

	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()

	request := &endly.OpenSession{
		Target: &endly.Resource{
			URL: "ssh://127.0.0.1:22/etc",
		},
	}
	response := srv.Run(context, request)
	assert.NotNil(t, response)
	assert.Nil(t, response.Error)
	assert.Equal(t, "ok", response.Status)

	os := context.OperatingSystem("127.0.0.1:22")
	assert.NotNil(t, os)
	assert.True(t, os.Name != "")
	assert.True(t, os.Version != "")
	assert.True(t, len(os.Path.SystemPath) > 0)
}
