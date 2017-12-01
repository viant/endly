package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"time"
)

func TestServer_Start(t *testing.T) {
	server := endly.NewServer("8432")
	go server.Start()

	time.Sleep(500 * time.Millisecond)
	request := &endly.Request{
		ServiceRequest: &endly.Nop{},
	}
	response := &endly.Response{
		Response: &endly.Nop{},
	}
	toolbox.RouteToService("post", "http://127.0.0.1:8432/v1/endly/service/nop/nop/", request, response)
	time.Sleep(1000 * time.Millisecond)
	assert.Equal(t, "", response.Error)
	assert.NotNil(t, response.Response)
	assert.Equal(t, "ok", response.Status)
	_, ok := response.Response.(*endly.Nop)
	assert.True(t, ok)

}
