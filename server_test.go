package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"testing"
	"time"
)

func TestServer_Start(t *testing.T) {

	server := endly.NewServer("8432")
	go server.Start()

	time.Sleep(500 * time.Millisecond)
	target := url.NewResource("ssh://127.0.0.1/etc")
	request := &endly.Request{
		ServiceRequest: endly.NewSimpleCommandRequest(target, "ls -al"),
	}
	response := &endly.Response{
		Response: &endly.CommandResponse{},
	}
	toolbox.RouteToService("post", "http://127.0.0.1:8432/v1/endly/service/exec/command/", request, response)
	time.Sleep(1000 * time.Millisecond)
	assert.Equal(t, "", response.Error)
	assert.NotNil(t, response.Response)
	assert.Equal(t, "ok", response.Status)
	commandInfo, ok := response.Response.(*endly.CommandResponse)
	assert.True(t, ok)
	if ok {
		assert.True(t, len(commandInfo.Stdout(0)) > 0)
	}

}
