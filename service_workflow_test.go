package endly_test

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/viant/endly"
	"github.com/viant/toolbox"

	"github.com/viant/toolbox/url"
	"errors"
	"testing"
	"github.com/stretchr/testify/assert"
	"os/exec"
)

func getServiceWithWorkflow(workflowURI string) (endly.Manager, endly.Service, error) {
	manager := endly.NewManager()
	service, err := manager.Service(endly.WorkflowServiceID)
	if err == nil {

		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowLoadRequest{
			Source: url.NewResource(workflowURI),
		})
		if response.Error != "" {
			return nil, nil, errors.New(response.Error)
		}
	}
	return manager, service, err
}



func TestWorkflowService_Run(t *testing.T) {

	exec.Command("rm", "-rf", "/tmp/endly/test/workflow/dsunit").CombinedOutput()
	toolbox.CreateDirIfNotExist("/tmp/endly/test/workflow/dsunit")
	manager, service, err := getServiceWithWorkflow("test/workflow/dsunit/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
			Name:"workflow",
			Tasks:"prepare",
			Params:map[string]interface{}{
				"param1":1,
			},
		})
		assert.Equal(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)

		if assert.True(t, ok) {
			assert.NotNil(t, response)
			var dsunit = toolbox.AsMap(response.Data["dsunit"])
			var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
			assert.EqualValues(t, 3, len(records))

		}

		context = manager.NewContext(toolbox.NewContext())
		serviceResponse = service.Run(context, &endly.WorkflowRunRequest{
			Name:"workflow",
			Tasks:"*",
			Params:map[string]interface{}{
				"param1":1,
			},
			EnableLogging:true,
			LoggingDirectory:"/tmp/dsunit/",
		})
		assert.Equal(t, "", serviceResponse.Error)

		response, ok = serviceResponse.Response.(*endly.WorkflowRunResponse)
		assert.NotNil(t, response)
		var dsunit = toolbox.AsMap(response.Data["dsunit"])
		var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
		assert.EqualValues(t, 0, len(records)) //validate task shift elements from USER_ACCCOUNT array.

	}
}

