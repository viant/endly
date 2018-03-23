package server

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"time"
)

func TestServer_Start(t *testing.T) {
	server := New("8432")
	go server.Start()

	time.Sleep(500 * time.Millisecond)
	request := &Request{
		ServiceRequest: &endly.NopRequest{},
	}
	response := &Response{
		Response: &endly.NopRequest{},
	}
	toolbox.RouteToService("post", "http://127.0.0.1:8432/v1/endly/service/nop/nop/", request, response)
	time.Sleep(1000 * time.Millisecond)
	assert.Equal(t, "", response.Error)
	assert.NotNil(t, response.Response)
	assert.Equal(t, "ok", response.Status)
	_, ok := response.Response.(*endly.NopRequest)
	assert.True(t, ok)

}
