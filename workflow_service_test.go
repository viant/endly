package endly_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"time"
)

func getServiceWithWorkflow(path string) (endly.Manager, endly.Service, error) {
	manager := endly.NewManager()
	service, err := manager.Service(endly.WorkflowServiceId)

	if err == nil {
		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowLoadRequest{
			Source: endly.NewFileResource("test/workflow/simple.csv"),
		})
		if response.Error != "" {
			return nil, nil, errors.New(response.Error)
		}
	}
	return manager, service, err
}

func TestRunWorfklow(t *testing.T) {

	go StartTestServer("8765")
	time.Sleep(500 * time.Millisecond)

	manager, service, err := getServiceWithWorkflow("test/workflow/simple.csv")
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, manager)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.WorkflowRunRequest{
		Name: "simple",
		Params: map[string]interface{}{
			"port": "8765",
		},
	})
	assert.Equal(t, "", response.Error)
	serviceResponse, ok := response.Response.(*endly.RunWorkflowRunResponse)
	assert.True(t, ok)
	assert.NotNil(t, serviceResponse)

}
