package endly_test
	//
	//import (
	//	"errors"
	//	"github.com/stretchr/testify/assert"
	//	"github.com/viant/endly"
	//	"github.com/viant/toolbox"
	//	"github.com/viant/toolbox/url"
	//	"os"
	//	"path"
	//	"strings"
	//	"testing"
	//	"time"
	//)
	//
	//func getServiceWithWorkflow(paths ...string) (endly.Manager, endly.Service, error) {
	//	manager := endly.NewManager()
	//	service, err := manager.Service(endly.WorkflowServiceID)
	//	if err == nil {
	//		for _, workflowPath := range paths {
	//			context := manager.NewContext(toolbox.NewContext())
	//			response := service.Run(context, &endly.WorkflowLoadRequest{
	//				Source: url.NewResource(workflowPath),
	//			})
	//			if response.Error != "" {
	//				return nil, nil, errors.New(response.Error)
	//			}
	//		}
	//	}
	//	return manager, service, err
	//}
	//
	//func TestRunWorkflow(t *testing.T) {
	//	go StartTestServer("8765")
	//	time.Sleep(500 * time.Millisecond)
	//	manager, service, err := getServiceWithWorkflow("test/workflow/simple.csv", "test/workflow/simple_call.csv")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	assert.NotNil(t, manager)
	//	assert.NotNil(t, service)
	//
	//	{
	//		context := manager.NewContext(toolbox.NewContext())
	//		response := service.Run(context, &endly.WorkflowRunRequest{
	//			Name: "simple",
	//			Params: map[string]interface{}{
	//				"port": "8765",
	//			},
	//		})
	//		assert.Equal(t, "", response.Error)
	//		serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//		assert.True(t, ok)
	//		assert.NotNil(t, serviceResponse)
	//	}
	//
	//	{
	//		context := manager.NewContext(toolbox.NewContext())
	//		response := service.Run(context, &endly.WorkflowRunRequest{
	//			Name: "simple_call",
	//			Params: map[string]interface{}{
	//				"port": "8765",
	//			},
	//		})
	//		assert.Equal(t, "", response.Error)
	//		serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//		assert.True(t, ok)
	//		assert.NotNil(t, serviceResponse)
	//	}
	//}
	//
	//func TestRunWorkflowMysql(t *testing.T) {
	//
	//	manager, service, err := getServiceWithWorkflow("workflow/dockerized_mysql.csv")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	assert.NotNil(t, manager)
	//	assert.NotNil(t, service)
	//
	//	targetHostCredential := path.Join(os.Getenv("HOME"), "/secret/scp.json")
	//	mysqlCredential := path.Join(os.Getenv("HOME"), "secret/mysql.json")
	//
	//	if toolbox.FileExists(mysqlCredential) {
	//
	//		{ //start docker
	//
	//			context := manager.NewContext(toolbox.NewContext())
	//			response := service.Run(context, &endly.WorkflowRunRequest{
	//				Name: "dockerized_mysql",
	//				Params: map[string]interface{}{
	//					"url":                 "scp://127.0.0.1/",
	//					"credential":          targetHostCredential,
	//					"mysqlCredential":     mysqlCredential,
	//					"stopSystemMysql":     true,
	//					"configUrl":           url.NewResource("test/docker/my.cnf").URL,
	//					"configURLCredential": path.Join(os.Getenv("HOME"), "/secret/scp.json"),
	//					"serviceInstanceName": "dockerizedMysql1",
	//				},
	//				Tasks: "start",
	//			})
	//
	//			if assert.Equal(t, "", response.Error) {
	//				serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//				assert.True(t, ok)
	//				assert.NotNil(t, serviceResponse)
	//			}
	//
	//		}
	//
	//		{ //start docker
	//
	//			context := manager.NewContext(toolbox.NewContext())
	//			response := service.Run(context, &endly.WorkflowRunRequest{
	//				Name: "dockerized_mysql",
	//				Params: map[string]interface{}{
	//					"url":                 "scp://127.0.0.1/",
	//					"credential":          targetHostCredential,
	//					"mysqlCredential":     mysqlCredential,
	//					"stopSystemMysql":     true,
	//					"configUrl":           url.NewResource("test/docker/my.cnf").URL,
	//					"configURLCredential": path.Join(os.Getenv("HOME"), "/secret/scp.json"),
	//					"serviceInstanceName": "dockerizedMysql1",
	//				},
	//				Tasks: "stop",
	//			})
	//
	//			if assert.Equal(t, "", response.Error) {
	//				serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//				assert.True(t, ok)
	//				assert.NotNil(t, serviceResponse)
	//			}
	//
	//		}
	//	}
	//
	//}
	//
	//func TestRunWorkflowAerospike(t *testing.T) {
	//
	//	manager, service, err := getServiceWithWorkflow("workflow/dockerized_aerospike.csv")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	assert.NotNil(t, manager)
	//	assert.NotNil(t, service)
	//	credential := path.Join(os.Getenv("HOME"), "secret/scp.json")
	//	if toolbox.FileExists(credential) {
	//		aerospikeConfigUrl := url.NewResource("test/workflow/aerospike.conf").URL
	//
	//		context := manager.NewContext(toolbox.NewContext())
	//		response := service.Run(context, &endly.WorkflowRunRequest{
	//			Name: "dockerized_aerospike",
	//			Params: map[string]interface{}{
	//				"url":                 "scp://127.0.0.1/",
	//				"credential":          credential,
	//				"configUrl":           aerospikeConfigUrl,
	//				"confifUrlCredential": "",
	//				"serviceInstanceName": "dockerizedAerospike1",
	//			},
	//			Tasks: "start",
	//		})
	//		if assert.Equal(t, "", response.Error) {
	//			serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//			assert.True(t, ok)
	//			assert.NotNil(t, serviceResponse)
	//		}
	//
	//		response = service.Run(context, &endly.WorkflowRunRequest{
	//			Name: "dockerized_aerospike",
	//			Params: map[string]interface{}{
	//				"url":                 "scp://127.0.0.1/",
	//				"credential":          credential,
	//				"configUrl":           aerospikeConfigUrl,
	//				"confifUrlCredential": "",
	//				"serviceInstanceName": "dockerizedAerospike1",
	//			},
	//			Tasks: "stop",
	//		})
	//		if assert.Equal(t, "", response.Error) {
	//			serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//			assert.True(t, ok)
	//			assert.NotNil(t, serviceResponse)
	//		}
	//	}
	//
	//}
	//
	//func TestRunWorfklowVCMavenwBuild(t *testing.T) {
	//
	//	manager, service, err := getServiceWithWorkflow("workflow/vc_maven_build.csv")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	assert.NotNil(t, manager)
	//	assert.NotNil(t, service)
	//	credential := path.Join(os.Getenv("HOME"), "secret/scp.json")
	//	if toolbox.FileExists(credential) {
	//		baseSvnUrlFile := path.Join(os.Getenv("HOME"), "baseSvnUrl")
	//		if toolbox.FileExists(baseSvnUrlFile) {
	//			baseSvnUrl, err := url.NewResource(path.Join(os.Getenv("HOME"), "baseSvnUrl")).DownloadText()
	//			baseSvnUrl = strings.Trim(baseSvnUrl, " \r\n")
	//			assert.Nil(t, err)
	//			context := manager.NewContext(toolbox.NewContext())
	//			response := service.Run(context, &endly.WorkflowRunRequest{
	//				Name: "vc_maven_build",
	//				Params: map[string]interface{}{
	//					"jdkVersion":           "1.7",
	//					"originUrl":            baseSvnUrl + "/common",
	//					"originCredential":     path.Join(os.Getenv("HOME"), "/secret/svn_ci.json"),
	//					"originType":           "svn",
	//					"targetUrl":            "file:///tmp/ci_common",
	//					"targetHostCredential": "",
	//					"buildGoal":            "install",
	//					"buildArgs":            "-Dmvn.test.skip",
	//				},
	//			})
	//			if assert.Equal(t, "", response.Error) {
	//				serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//				assert.True(t, ok)
	//				assert.NotNil(t, serviceResponse)
	//
	//			}
	//		}
	//
	//	}
	//
	//}
	//
	//func TestRunWorfklowTomcatApp(t *testing.T) {
	//
	//	manager, service, err := getServiceWithWorkflow("workflow/tomcat.csv")
	//	if !assert.Nil(t, err) {
	//		return
	//	}
	//	assert.NotNil(t, manager)
	//	assert.NotNil(t, service)
	//	targetHostCredential := path.Join(os.Getenv("HOME"), "secret/scp.json")
	//
	//	if toolbox.FileExists(targetHostCredential) {
	//		configUrl := url.NewResource("test/workflow/tomcat-server.xml").URL
	//
	//		baseSvnUrlFile := path.Join(os.Getenv("HOME"), "baseSvnUrl")
	//		if toolbox.FileExists(baseSvnUrlFile) {
	//			baseSvnUrl, err := url.NewResource(path.Join(os.Getenv("HOME"), "baseSvnUrl")).DownloadText()
	//			baseSvnUrl = strings.Trim(baseSvnUrl, " \r\n")
	//			assert.Nil(t, err)
	//			context := manager.NewContext(toolbox.NewContext())
	//
	//			{
	//				response := service.Run(context, &endly.WorkflowRunRequest{
	//					Name: "tomcat",
	//					Params: map[string]interface{}{
	//						"targetHost":           "127.0.0.1",
	//						"targetHostCredential": targetHostCredential,
	//						"appDirectory":         "/tmp/app1",
	//						"configUrl":            configUrl,
	//						"configURLCredential":  targetHostCredential,
	//						"tomcatPort":           "8881",
	//						"forceDeploy":          true,
	//					},
	//					Tasks: "install",
	//				})
	//				if assert.Equal(t, "", response.Error) {
	//					serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//					assert.True(t, ok)
	//					assert.NotNil(t, serviceResponse)
	//
	//				}
	//			}
	//			{
	//				response := service.Run(context, &endly.WorkflowRunRequest{
	//					Name: "tomcat",
	//					Params: map[string]interface{}{
	//						"jdkVersion":           "1.7",
	//						"targetHost":           "127.0.0.1",
	//						"targetHostCredential": targetHostCredential,
	//						"appDirectory":         "/tmp/app1",
	//					},
	//					Tasks: "start",
	//				})
	//				if assert.Equal(t, "", response.Error) {
	//					serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//					assert.True(t, ok)
	//					assert.NotNil(t, serviceResponse)
	//
	//				}
	//			}
	//
	//			time.Sleep(2 * time.Second)
	//			{
	//				response := service.Run(context, &endly.WorkflowRunRequest{
	//					Name: "tomcat",
	//					Params: map[string]interface{}{
	//						"jdkVersion":           "1.7",
	//						"targetHost":           "127.0.0.1",
	//						"targetHostCredential": targetHostCredential,
	//						"appDirectory":         "/tmp/app1",
	//					},
	//					Tasks: "stop",
	//				})
	//				if assert.Equal(t, "", response.Error) {
	//					serviceResponse, ok := response.Response.(*endly.WorkflowRunResponse)
	//					assert.True(t, ok)
	//					assert.NotNil(t, serviceResponse)
	//
	//				}
	//			}
	//		}
	//
	//	}
	//
	//}
