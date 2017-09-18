package endly_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"time"
	"path"
	"os"
)

func getServiceWithWorkflow(path string) (endly.Manager, endly.Service, error) {
	manager := endly.NewManager()
	service, err := manager.Service(endly.WorkflowServiceId)

	if err == nil {
		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowLoadRequest{
			Source: endly.NewFileResource(path),
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
	serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	assert.True(t, ok)
	assert.NotNil(t, serviceResponse)

}

func TestRunWorfklowMysql(t *testing.T) {

	manager, service, err := getServiceWithWorkflow("workflow/dockerized_mysql.csv")
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, manager)
	assert.NotNil(t, service)

	{
		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowRunRequest{
			Name: "dockerized_mysql",
			Params: map[string]interface{}{
				"url":        "scp://127.0.0.1/",
				"credential": path.Join(os.Getenv("HOME"), "/secret/scp.json"),
			},
			Tasks:map[string]string{
				"system_stop_mysql":"0,1",
				"system_start_docker":"0",
			},

		})
		if assert.Equal(t, "", response.Error) {
			serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
			assert.True(t, ok)
			assert.NotNil(t, serviceResponse)

			assert.Equal(t, "system_stop_mysql", serviceResponse.TasksActivities[0].Task)
			assert.Equal(t, "Does not match run criteria: $params.stopSystemMysql:true", serviceResponse.TasksActivities[0].Skipped)

			if len(serviceResponse.TasksActivities[0].ServiceActivities) > 0 {
				assert.Equal(t, "status", serviceResponse.TasksActivities[0].ServiceActivities[0].Action)
				assert.Equal(t, "", serviceResponse.TasksActivities[0].ServiceActivities[0].Skipped)

				assert.Equal(t, "stop", serviceResponse.TasksActivities[0].ServiceActivities[1].Action)
				assert.Equal(t, "", serviceResponse.TasksActivities[0].ServiceActivities[1].Skipped)
			}
		}
	}

	credential := path.Join(os.Getenv("HOME"), "secret/mysql.json")
	if toolbox.FileExists(credential) {
		context := manager.NewContext(toolbox.NewContext())
		response := service.Run(context, &endly.WorkflowRunRequest{
			Name: "dockerized_mysql",
			Params: map[string]interface{}{
				"url":             "scp://127.0.0.1/",
				"credential":      path.Join(os.Getenv("HOME"), "/secret/scp.json"),
				"stopSystemMysql": true,
				"mycnfUrl": endly.NewFileResource("test/docker/my.cnf").URL,
				"mycnfUrlCredential":"",
				"serviceInstanceName":"dockerizedMysql1",
			},
		})
		if assert.Equal(t, "", response.Error) {
			serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
			assert.True(t, ok)
			assert.NotNil(t, serviceResponse)

			assert.Equal(t, "system_stop_mysql", serviceResponse.TasksActivities[0].Task)
			assert.Equal(t, "", serviceResponse.TasksActivities[0].Skipped)

			assert.Equal(t, "status", serviceResponse.TasksActivities[0].ServiceActivities[0].Action)
			assert.Equal(t, "", serviceResponse.TasksActivities[0].ServiceActivities[0].Skipped)
			assert.Equal(t, "stop", serviceResponse.TasksActivities[0].ServiceActivities[1].Action)

			assert.Equal(t, "status", serviceResponse.TasksActivities[1].ServiceActivities[0].Action)
			assert.Equal(t, "start", serviceResponse.TasksActivities[1].ServiceActivities[1].Action)
		}


	}

}
