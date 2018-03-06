package workflow_test

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/endpoint/http"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/viant/endly/cli"

	_ "github.com/viant/endly/cloud/ec2"
	_ "github.com/viant/endly/cloud/gce"
	_ "github.com/viant/endly/endpoint/http"

	_ "github.com/viant/endly/testing/dsunit"
	_ "github.com/viant/endly/testing/log"
	_ "github.com/viant/endly/testing/validator"

	_ "github.com/viant/endly/runner/http"
	_ "github.com/viant/endly/runner/selenium"

	_ "github.com/viant/endly/deployment/build"
	_ "github.com/viant/endly/deployment/deploy"
	_ "github.com/viant/endly/deployment/sdk"
	_ "github.com/viant/endly/deployment/vc"

	_ "github.com/viant/endly/system/daemon"
	_ "github.com/viant/endly/system/docker"
	_ "github.com/viant/endly/system/exec"
	_ "github.com/viant/endly/system/process"
	_ "github.com/viant/endly/system/storage"

	"github.com/viant/endly/workflow"
)

func getServiceWithWorkflow(workflowURI string) (endly.Manager, endly.Service, error) {
	manager := endly.New()
	service, err := manager.Service(workflow.ServiceID)
	if err == nil {

		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &workflow.LoadRequest{
			Source: url.NewResource(workflowURI),
		})
		if response.Error != "" {
			return nil, nil, errors.New(response.Error)
		}

	}
	return manager, service, err
}

func getServiceWithWorkflowContext(workflowURI string) (*endly.Context, endly.Service, error) {
	manager := endly.New()
	service, err := manager.Service(workflow.ServiceID)
	context := manager.NewContext(toolbox.NewContext())

	if err == nil {
		response := service.Run(context, &workflow.LoadRequest{
			Source: url.NewResource(workflowURI),
		})
		if response.Error != "" {
			return nil, nil, errors.New(response.Error)
		}
		if workflowLoadResponse, ok := response.Response.(*workflow.LoadResponse); ok {
			context.Workflows.Push(workflowLoadResponse.Workflow)
		} else {
			fmt.Printf("unexpected response: %T\n", response.Response)
		}
	}
	return context, service, err
}

func TestWorkflowService_SwitchAction(t *testing.T) {
	context, service, err := getServiceWithWorkflowContext("test/nop/workflow.csv")
	assert.Nil(t, err)

	request := &workflow.SwitchRequest{
		SourceKey: "run",
		Cases: []*workflow.SwitchCase{
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
		Default: &workflow.SwitchCase{
			ActionRequest: &endly.ActionRequest{
				Service: "nop",
				Action:  "parrot",
				Request: map[string]interface{}{
					"In": map[string]interface{}{"r": "test 3"},
				},
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
		response := toolbox.AsMap(serviceResponse.Response)
		assert.EqualValues(t, testCase.Expected, response["r"])
	}

}

func TestWorkflowService_RunDsUnitWorkflow(t *testing.T) {

	manager, service, err := getServiceWithWorkflow("test/dsunit/workflow.csv")
	if assert.Nil(t, err) {

		{
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:  "workflow",
				Tasks: "prepare",
				Params: map[string]interface{}{
					"param1": 1,
				},
				EnableLogging:    true,
				LoggingDirectory: "logs",
			})

			if !assert.NotNil(t, serviceResponse) {
				return
			}

			assert.Equal(t, "", serviceResponse.Error)
			response, ok := serviceResponse.Response.(*workflow.RunResponse)

			if assert.True(t, ok) {
				if assert.NotNil(t, response) {
					if assert.True(t, len(response.Data) > 0) {
						return
					}

					dsUnit, ok := response.Data["dsunit"]
					if !ok {
						assert.Fail(t, fmt.Sprintf("expected dsunit key dsunit: %v", response.Data))
						return
					}
					var dsunit = toolbox.AsMap(dsUnit)
					var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
					assert.EqualValues(t, 3, len(records))

				}

			}
		}

		{
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:  "workflow",
				Tasks: "*",
				Params: map[string]interface{}{
					"param1": 1,
				},
				EnableLogging:    true,
				LoggingDirectory: "/tmp/logs",
			})
			assert.Equal(t, "", serviceResponse.Error)

			response, ok := serviceResponse.Response.(*workflow.RunResponse)
			assert.True(t, ok)
			assert.NotNil(t, response)
			var dsunit = toolbox.AsMap(response.Data["dsunit"])
			var records = toolbox.AsSlice(dsunit["USER_ACCOUNT"])
			assert.EqualValues(t, 0, len(records)) //validate task shift elements from USER_ACCCOUNT array.

		}
	}
}

func TestWorkflowService_OnErrorTask(t *testing.T) {

	manager, service, _ := getServiceWithWorkflow("test/recover/workflow.csv")

	context := manager.NewContext(toolbox.NewContext())
	serviceResponse := service.Run(context, &workflow.RunRequest{
		Name:             "recover",
		Tasks:            "fail",
		Params:           map[string]interface{}{},
		EnableLogging:    false,
		LoggingDirectory: "logs",
	})

	assert.EqualValues(t, "", serviceResponse.Error)
	response, ok := serviceResponse.Response.(*workflow.RunResponse)
	if assert.True(t, ok) {
		errorCaught := toolbox.AsString(response.Data["errorCaught"])
		assert.True(t, strings.Contains(errorCaught, "this is test error "))
	}
}

func TestWorkflowService_RunHttpWorkflow(t *testing.T) {

	baseDir := toolbox.CallerDirectory(3)
	err := http.StartServer(8113, &http.HTTPServerTrips{
		IndexKeys:     []string{http.MethodKey, http.URLKey, http.BodyKey, http.CookieKey, http.ContentTypeKey},
		BaseDirectory: path.Join(baseDir, "test/endpoint"),
	})

	if !assert.Nil(t, err) {
		return
	}

	manager, service, err := getServiceWithWorkflow("test/http/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &workflow.RunRequest{
			Name:  "http_workflow",
			Tasks: "*",
			Params: map[string]interface{}{
				"appServer": "http://127.0.0.1:8113",
			},
			PublishParameters: true,
			EnableLogging:     true,
			LoggingDirectory:  "logs",
		})
		assert.EqualValues(t, "", serviceResponse.Error)
		response, ok := serviceResponse.Response.(*workflow.RunResponse)
		if assert.True(t, ok) {

			responses, ok := response.Data["httpResponses"]
			if assert.True(t, ok) {
				httpResponses := toolbox.AsSlice(responses)
				assert.EqualValues(t, 3, len(httpResponses))
				for _, item := range httpResponses {
					httpResponse := toolbox.AsMap(item)
					assert.EqualValues(t, 200, httpResponse["Code"])
				}
			}
		}
	}
}

func TestWorkflowService_RunLifeCycle(t *testing.T) {

	manager, service, err := getServiceWithWorkflow("test/lifecycle/workflow.csv")
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())
		serviceResponse := service.Run(context, &workflow.RunRequest{
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
			response, ok := serviceResponse.Response.(*workflow.RunResponse)
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

		manager, service, err := getServiceWithWorkflow("test/broken/broken1.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:              "broken1",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "broken1"), serviceResponse.Error)
		}
	}
	{
		//unsupported action error

		manager, service, err := getServiceWithWorkflow("test/broken/broken2.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:              "broken2",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "unknown nop.aaa service action at workflow.run"), serviceResponse.Error)
		}
	}

	{
		//unsupported action error

		manager, service, err := getServiceWithWorkflow("test/broken/broken2.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:              "broken2",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "unknown nop.aaa service action at workflow.run"), serviceResponse.Error)
		}
	}

	{
		//unsupported service error

		manager, service, err := getServiceWithWorkflow("test/broken/broken3.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
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

		manager, service, err := getServiceWithWorkflow("test/broken/broken4.csv")
		if assert.Nil(t, err) {
			context := manager.NewContext(toolbox.NewContext())
			serviceResponse := service.Run(context, &workflow.RunRequest{
				Name:              "broken4",
				Tasks:             "*",
				Params:            map[string]interface{}{},
				PublishParameters: true,
			})
			assert.EqualValues(t, true, strings.Contains(serviceResponse.Error, "failed to load workflow"), serviceResponse.Error)
		}
	}
}

func Test_WorkflowSwitchRequest_Validate(t *testing.T) {
	{
		request := &workflow.SwitchRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &workflow.SwitchRequest{
			SourceKey: "abc",
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &workflow.SwitchRequest{
			SourceKey: "abc",
			Cases: []*workflow.SwitchCase{
				{},
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := &workflow.SwitchRequest{
			SourceKey: "abc",
			Cases: []*workflow.SwitchCase{
				{
					Value: "123",
				},
			},
		}
		assert.Nil(t, request.Validate())
	}
}
