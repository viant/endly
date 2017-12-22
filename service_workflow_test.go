package endly_test

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/viant/endly"
	"github.com/viant/toolbox"

	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox/url"
	"os/exec"
	"path"
	"strings"
	"testing"
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

		{
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:  "workflow",
				Tasks: "prepare",
				Params: map[string]interface{}{
					"param1": 1,
				},
				EnableLogging:    true,
				LoggingDirectory: "/tmp/logs",
			})
			assert.Equal(t, "", serviceResponse.Error)
			response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)

			if assert.True(t, ok) {
				assert.NotNil(t, response)
				var dsunit = toolbox.AsMap(response.Data["dsunit"])
				var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
				assert.EqualValues(t, 3, len(records))

			}
		}

		{
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:  "workflow",
				Tasks: "*",
				Params: map[string]interface{}{
					"param1": 1,
				},
				EnableLogging:    true,
				LoggingDirectory: "/tmp/logs",
			})
			assert.Equal(t, "", serviceResponse.Error)

			response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)
			assert.True(t, ok)
			assert.NotNil(t, response)
			var dsunit = toolbox.AsMap(response.Data["dsunit"])
			var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
			assert.EqualValues(t, 0, len(records)) //validate task shift elements from USER_ACCCOUNT array.

		}
	}
}

func TestWorkflowService_RunHttpWorkflow(t *testing.T) {

	baseDir := toolbox.CallerDirectory(3)
	err := endly.StartHTTPServer(8113, &endly.HTTPServerTrips{
		IndexKeys:     []string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
		BaseDirectory: path.Join(baseDir, "test/http/runner/http_workflow"),
	})

	if !assert.Nil(t, err) {
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
			EnableLogging:     true,
			LoggingDirectory:  "/tmp/http/",
		})
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)
		if assert.True(t, ok) {

			httpResponses := toolbox.AsSlice(response.Data["httpResponses"])
			assert.EqualValues(t, 3, len(httpResponses))
			for _, item := range httpResponses {
				httpResponse := toolbox.AsMap(item)
				assert.EqualValues(t, 200, httpResponse["Code"])
			}
		}
	}
}

func TestWorkflowService_RunLifeCycle(t *testing.T) {

	manager, service, err := getServiceWithWorkflow("test/workflow/lifecycle/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
			Name:  "lifecycle",
			Tasks: "*",
			Params: map[string]interface{}{
				"object": map[string]interface{}{
					"key1": 1,
					"key2": "abc",
				},
			},
			PublishParameters: true,
			EnableLogging:     true,
			LoggingDirectory:  "logs",
		})

		if assert.EqualValues(t, "", serviceResponse.Error) {
			response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)
			if assert.True(t, ok) {
				assert.EqualValues(t, 2, response.Data["testPassed"])
				var anArray = toolbox.AsSlice(response.Data["array"])
				assert.EqualValues(t, 2, anArray[0])
				assert.EqualValues(t, 3, response.Data["counter"])
				var anObject = toolbox.AsMap(response.Data["object"])
				assert.EqualValues(t, 1, anObject["key1"])
				assert.EqualValues(t, "200", anObject["shift"])
			}
		}
	}
}

func TestWorkflowService_RunBroken(t *testing.T) {

	{
		//request empty error

		manager, service, err := getServiceWithWorkflow("test/workflow/broken/broken1.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:              "broken1",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "failed to evaluate request"), serviceResponse.Error)
		}
	}
	{
		//unsupported action error

		manager, service, err := getServiceWithWorkflow("test/workflow/broken/broken2.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:              "broken2",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "unsupported action: aaa"), serviceResponse.Error)
		}
	}

	{
		//unsupported action error

		manager, service, err := getServiceWithWorkflow("test/workflow/broken/broken2.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:              "broken2",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "unsupported action: aaa"), serviceResponse.Error)
		}
	}

	{
		//unsupported service error

		manager, service, err := getServiceWithWorkflow("test/workflow/broken/broken3.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:              "broken3",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "failed to lookup service: 'aaa'"), serviceResponse.Error)
		}
	}

	{
		//calling invalid workflow

		manager, service, err := getServiceWithWorkflow("test/workflow/broken/broken4.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
				Name:              "broken4",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "failed to load workflow"), serviceResponse.Error)
		}
	}
}
