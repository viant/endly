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
	"path"
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

func TestWorkflowService_RunDsUnitWorkflow(t *testing.T) {

	exec.Command("rm", "-rf", "/tmp/endly/test/workflow/dsunit").CombinedOutput()
	toolbox.CreateDirIfNotExist("/tmp/endly/test/workflow/dsunit")
	manager, service, err := getServiceWithWorkflow("test/workflow/dsunit/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
			Name:  "workflow",
			Tasks: "prepare",
			Params: map[string]interface{}{
				"param1": 1,
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
			Name:  "workflow",
			Tasks: "*",
			Params: map[string]interface{}{
				"param1": 1,
			},
			EnableLogging:    true,
			LoggingDirectory: "/tmp/dsunit/",
		})
		assert.Equal(t, "", serviceResponse.Error)

		response, ok = serviceResponse.Response.(*endly.WorkflowRunResponse)
		assert.NotNil(t, response)
		var dsunit = toolbox.AsMap(response.Data["dsunit"])
		var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
		assert.EqualValues(t, 0, len(records)) //validate task shift elements from USER_ACCCOUNT array.

	}
}

func TestWorkflowService_RunHttpWorkflow(t *testing.T) {

	baseDir := toolbox.CallerDirectory(3)
	err := endly.StartHttpServer(8113, &endly.HttpServerTrips{
		IndexKeys:     []string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
		BaseDirectory: path.Join(baseDir, "test/http/runner/http_workflow"),
	})

	if ! assert.Nil(t, err) {
		return
	}

	manager, service, err := getServiceWithWorkflow("test/workflow/http/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
			Name:  "http_workflow",
			Tasks: "*",
			Params: map[string]interface{}{
				"appServer": "http://127.0.0.1:8113",
			},
			PublishParameters: true,
		})
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)
		if assert.True(t, ok) {

			httpResponses := toolbox.AsSlice(response.Data["httpResponses"])
			assert.EqualValues(t, 3,len(httpResponses))
			for _, item := range httpResponses {
				httpResponse := toolbox.AsMap(item)
				assert.EqualValues(t, 200, httpResponse["Code"])
			}
		}
	}
}
