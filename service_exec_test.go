package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"testing"
)

func TestNewExecService(t *testing.T) {

	manager := endly.NewManager()
	srv, err := manager.Service(endly.ExecServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, srv)

	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()
	request := &endly.OpenSessionRequest{
		Target: &url.Resource{
			URL: "ssh://127.0.0.1:22/etc",
		},
	}

	response := srv.Run(context, request)
	assert.NotNil(t, response)
	assert.Equal(t, "", response.Error)
	assert.Equal(t, "ok", response.Status)

	os := context.OperatingSystem("127.0.0.1:22")
	assert.NotNil(t, os)
	assert.True(t, os.Name != "")
	assert.True(t, os.Version != "")
	assert.True(t, len(os.Path.SystemPath) > 0)
}
