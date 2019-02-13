package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	_ "github.com/viant/endly/system/kubernetes/apps"
	_ "github.com/viant/endly/system/kubernetes/core"
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"log"
	"path"
	"testing"
)

func TestNew(t *testing.T) {
	service := New()
	assert.NotNil(t, service)
}

func TestService_Get(t *testing.T) {
	service := New()

	for _, k := range shared.MetaTypes() {
		fmt.Printf("%v\n", k)
	}
	manager := endly.New()
	context := manager.NewContext(nil)

	{
		getRequest := &GetRequest{}
		getRequest.Kind = "*"
		getRequest.Name = "pi-kgmsk"
		assert.Nil(t, getRequest.Init())

		resp, err := service.Get(context, getRequest)
		if !assert.Nil(t, err) {
			log.Fatal(err)
		}
		assert.NotNil(t, resp)
		toolbox.DumpIndent(resp, true)
	}

	//{
	//	getRequest := &GetRequest{}
	//	getRequest.Kind = "pod"
	//	assert.Nil(t, getRequest.Init())
	//
	//	resp, err := service.Get(context, getRequest)
	//	if !assert.Nil(t, err) {
	//		log.Fatal(err)
	//	}
	//	assert.NotNil(t, resp)
	//	toolbox.DumpIndent(resp, true)
	//}

}

func TestService_Create(t *testing.T) {

	service := New()
	manager := endly.New()
	context := manager.NewContext(nil)

	shared.Init(context, map[string]interface{}{
		"k1": "default",
	})

	{
		parent := toolbox.CallerDirectory(3)
		createRequest := &ApplyRequest{
			Resource: url.NewResource(path.Join(parent, "test/env.yaml")),
		}
		assert.Nil(t, createRequest.Init())

		resp, err := service.Apply(context, createRequest)
		if !assert.Nil(t, err) {
			log.Fatal(err)
		}
		assert.NotNil(t, resp)
		toolbox.DumpIndent(resp, true)
	}

}
