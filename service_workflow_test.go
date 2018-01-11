package endly_test

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/viant/endly"
	"github.com/viant/toolbox"

	"errors"
	"fmt"
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

func getServiceWithWorkflowContext(workflowURI string) (*endly.Context, endly.Service, error) {
	manager := endly.NewManager()
	service, err := manager.Service(endly.WorkflowServiceID)
	context := manager.NewContext(toolbox.NewContext())

	if err == nil {
		response := service.Run(context, &endly.WorkflowLoadRequest{
			Source: url.NewResource(workflowURI),
		})
		if response.Error != "" {
			return nil, nil, errors.New(response.Error)
		}
		if workflowLoadResponse, ok := response.Response.(*endly.WorkflowLoadResponse); ok {
			context.Workflows.Push(workflowLoadResponse.Workflow)
		} else {
			fmt.Printf("unexpected reponse: %T\n", response.Response)
		}
	}
	return context, service, err
}

func TestWorkflowService_RepeatTask(t *testing.T) {
	context, service, err := getServiceWithWorkflowContext("test/workflow/nop/workflow.csv")
	assert.Nil(t, err)
	serviceResponse := service.Run(context, &endly.WorkflowRepeatTaskRequest{
		Task: "task1",
		Repeatable: &endly.Repeatable{
			Repeat:      2,
			SleepTimeMs: 1,
		},
	})
	assert.EqualValues(t, "", serviceResponse.Error)

	response, ok := serviceResponse.Response.(*endly.WorkflowRepeatTaskResponse)
	if assert.True(t, ok, fmt.Sprintf("expected %T but had %T", &endly.WorkflowRepeatTaskResponse{}, serviceResponse.Response)) {
		assert.EqualValues(t, 2, response.Repeated)
	}

}

func TestWorkflowService_RepeatAction(t *testing.T) {
	context, service, err := getServiceWithWorkflowContext("test/workflow/nop/workflow.csv")
	assert.Nil(t, err)
	serviceResponse := service.Run(context, &endly.WorkflowRepeatActionRequest{
		ActionRequest: &endly.ActionRequest{
			Service: "nop",
			Action:  "nop",
			Request: map[string]interface{}{},
		},
		Repeatable: &endly.Repeatable{
			Repeat:      2,
			SleepTimeMs: 1,
		},
	})
	assert.EqualValues(t, "", serviceResponse.Error)

	response, ok := serviceResponse.Response.(*endly.WorkflowRepeatActionResponse)
	if assert.True(t, ok, fmt.Sprintf("expected %T but had %T", &endly.WorkflowRepeatActionResponse{}, serviceResponse.Response)) {
		assert.EqualValues(t, 2, response.Repeated)
	}
}

func TestWorkflowService_SwitchTask(t *testing.T) {
	context, service, err := getServiceWithWorkflowContext("test/workflow/nop/workflow.csv")
	assert.Nil(t, err)

	request := &endly.WorkflowSwitchTaskRequest{
		SourceKey: "goto",
		Cases: []*endly.WorkflowSwitchTaskCase{
			{
				Value: "task1",
				Task:  "task1",
			},
			{
				Value: "task2",
				Task:  "task2",
			},
		},
		DefaultTask: "task3",
	}

	var state = context.State()

	type useCase struct {
		SourceKey    string
		ExpectedTask string
	}

	for _, testCase := range []useCase{
		{
			SourceKey:    "task1",
			ExpectedTask: "task1",
		},
		{
			SourceKey:    "task2",
			ExpectedTask: "task2",
		},
		{
			SourceKey:    "unknown",
			ExpectedTask: "task3",
		},
	} {
		state.Put("goto", testCase.SourceKey)
		serviceResponse := service.Run(context, request)
		assert.EqualValues(t, "", serviceResponse.Error)
		response := serviceResponse.Response.(*endly.WorkflowSwitchTaskResponse)
		assert.EqualValues(t, testCase.ExpectedTask, response.Task)
	}

}

func TestWorkflowService_SwitchAction(t *testing.T) {
	context, service, err := getServiceWithWorkflowContext("test/workflow/nop/workflow.csv")
	assert.Nil(t, err)

	request := &endly.WorkflowSwitchActionRequest{
		SourceKey: "run",
		Cases: []*endly.WorkflowSwitchActionCase{
			{
				Value: "action1",
				ActionRequest: &endly.ActionRequest{
					Service: "nop",
					Action:  "parrot",
					Request: map[string]interface{}{
						"In": map[string]interface{}{"r": "test 1"},
					},
				},
			},
			{
				Value: "action2",
				ActionRequest: &endly.ActionRequest{
					Service: "nop",
					Action:  "parrot",
					Request: map[string]interface{}{
						"In": map[string]interface{}{"r": "test 2"},
					},
				},
			},
		},
		Default: &endly.ActionRequest{
			Service: "nop",
			Action:  "parrot",
			Request: map[string]interface{}{
				"In": map[string]interface{}{"r": "test 3"},
			},
		},
	}

	var state = context.State()

	type useCase struct {
		SourceKey string
		Expected  interface{}
	}

	for _, testCase := range []useCase{
		{
			SourceKey: "action1",
			Expected:  "test 1",
		},
		{
			SourceKey: "action2",
			Expected:  "test 2",
		},
		{
			SourceKey: "unknown",
			Expected:  "test 3",
		},
	} {
		state.Put("run", testCase.SourceKey)
		serviceResponse := service.Run(context, request)
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*endly.WorkflowSwitchActionResponse)
		if assert.True(t, ok) {
			assert.EqualValues(t, "parrot", response.Action)
			if assert.True(t, ok) {
				responseMap := toolbox.AsMap(response.Response)
				//fmt.Printf(" %T\n", response.Response)
				assert.EqualValues(t, testCase.Expected, responseMap["r"])
			}
		}
	}

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

func TestWorkflowService_OnErrorTask(t *testing.T) {

	manager, service, _ := getServiceWithWorkflow("test/workflow/recover/workflow.csv")

	context := manager.NewContext(toolbox.NewContext())
	serviceResponse := service.Run(context, &endly.WorkflowRunRequest{
		Name:             "recover",
		Tasks:            "fail",
		Params:           map[string]interface{}{},
		EnableLogging:    false,
		LoggingDirectory: "logs",
	})

	assert.EqualValues(t, "", serviceResponse.Error)
	response, ok := serviceResponse.Response.(*endly.WorkflowRunResponse)
	if assert.True(t, ok) {
		errorCaught := toolbox.AsString(response.Data["errorCaught"])
		assert.True(t, strings.Contains(errorCaught, "Fail this is test error at"))
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
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "failed to run action:Broken request was nil for nop.nop"), serviceResponse.Error)
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
