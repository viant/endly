package core

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	_ "github.com/viant/endly/system/kubernetes/apps"
	_ "github.com/viant/endly/system/kubernetes/core"
	"github.com/viant/toolbox"
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
}

func TestService_Get(t *testing.T) {
	service := New()

	manager := endly.New()
	context := manager.NewContext(nil)

	{
		getRequest := &GetRequest{}
		getRequest.Kind = "Pod"
		assert.Nil(t, getRequest.Init())

		resp, err := service.Get(context, getRequest)
		if !assert.Nil(t, err) {
			log.Fatal(err)
		}
		assert.NotNil(t, resp)
		toolbox.DumpIndent(resp, true)
	}

	{
		getRequest := &GetRequest{}
		getRequest.Name = "pod/hello-world-5b446dd74b-hphmg"
		assert.Nil(t, getRequest.Init())

		resp, err := service.Get(context, getRequest)
		if !assert.Nil(t, err) {
			log.Fatal(err)
		}
		assert.NotNil(t, resp)
		toolbox.DumpIndent(resp, true)
	}

}
