package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewManager(t *testing.T) {

	manager := endly.GetManager()
	context := manager.NewContext(toolbox.NewContext())
	manager.Register(newTestService())
	manager.RegisterCredentialFile("abc", "/Users/awitas/secret/abc.json")

	service, err := manager.Service("testService")
	assert.Nil(t, err)
	assert.NotNil(t, service)

	_, err = manager.Service("cc")
	assert.NotNil(t, err)

	file, err := manager.CredentialFile("abc")
	assert.Nil(t, err)
	assert.Equal(t, "/Users/awitas/secret/abc.json", file)

	_, err = manager.CredentialFile("cc")
	assert.NotNil(t, err)

	manager2, err := context.ServiceManager()
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

func (t *testService) Run(context *endly.Context, request interface{}) *endly.Response {
	return &endly.Response{}
}

func newTestService() endly.Service {
	var result = &testService{
		AbstractService: endly.NewAbstractService("testService"),
	}
	result.AbstractService.Service = result
	return result

}
