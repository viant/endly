package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
   _"github.com/viant/endly/static"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewManager(t *testing.T) {

	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	manager.Register(newTestService())

	service, err := manager.Service("testService")
	assert.Nil(t, err)
	assert.NotNil(t, service)

	_, err = manager.Service("cc")
	assert.NotNil(t, err)

	manager2, err := context.Manager()
	assert.Nil(t, err)
	assert.Equal(t, manager2, manager)

	{
		service, err := manager2.Service("testService")
		assert.Nil(t, err)
		assert.NotNil(t, service)

	}

	{
		service, err := context.Service("testService")
		assert.Nil(t, err)
		assert.NotNil(t, service)

	}

	{
		state := context.State()
		assert.NotNil(t, state)
		state.Put("k1", 1)
	}
	{
		state := context.State()
		assert.Equal(t, 1, state.GetInt("k1"))
	}

}

type testService struct {
	*endly.AbstractService
}

func (t *testService) Run(context *endly.Context, request interface{}) *endly.ServiceResponse {
	return &endly.ServiceResponse{}
}

func newTestService() endly.Service {
	var result = &testService{
		AbstractService: endly.NewAbstractService("testService"),
	}
	result.AbstractService.Service = result
	return result

}

func Test_ServiceRoutes(t *testing.T) {
	manager := endly.NewManager()
	var services = endly.Services(manager)
	var context = manager.NewContext(toolbox.NewContext())
	for _, service := range services {
		response := service.Run(context, struct{}{})
		assert.True(t, response.Error != "")
		for _, action := range service.Actions() {
			if route, err := service.ServiceActionRoute(action); err == nil {
				if route.Handler != nil {
					_, err := route.Handler(context, struct{}{})
					assert.NotNil(t, err)
				}
			}
		}
	}
}

func TestNewManager_Run(t *testing.T) {
	manager := endly.NewManager()

	{
		_, err := manager.Run(nil, &endly.NopParrotRequest{
			In: "Hello world",
		})
		if assert.Nil(t, err) {

		}
	}

	{
		_, err := manager.Run(nil, &struct{}{})
		if assert.NotNil(t, err) {

		}
	}
}

func Test_GetVersion(t *testing.T) {
	version := endly.GetVersion()
	assert.True(t, version != "")
}
