package selenium_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	endpoint "github.com/viant/endly/endpoint/http"
	runner "github.com/viant/endly/runner/selenium"
	"github.com/viant/endly/system/exec"
	tstorage "github.com/viant/endly/system/storage"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"testing"
)

const code = `
	package main
	import "fmt"

	func main() {
		fmt.Println("Hello WebDriver!\n")
	}
`

func TestSeleniumService_Start(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile)
	assert.Nil(t, err)
	var manager = endly.New()
	var useCases = []struct {
		baseDir     string
		DataURLs    []string
		DataPayload []byte
		target      *url.Resource
		request     *runner.StartRequest
		Pid         int
	}{
		{
			"test/start/inactive/darwin",
			[]string{
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-macos.tar.gz",
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-linux64.tar.gz",
				"http://selenium-release.storage.googleapis.com/3.4/selenium-server-standalone-3.4.0.jar",
				"scp://127.0.0.1:22/opt/selenium-server-standalone.jar",
			},
			[]byte("test"),
			url.NewResource("scp://127.0.0.1:22/", credentialFile),
			&runner.StartRequest{
				Target:     target,
				Sdk:        "jdk",
				SdkVersion: "1.8",
				Version:    "3.4",
				Port:       8117,
			},
			28811,
		},
		{
			"test/start/active/darwin",
			[]string{
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-macos.tar.gz",
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-linux64.tar.gz",
				"http://selenium-release.storage.googleapis.com/3.4/selenium-server-standalone-3.4.0.jar",
				"scp://127.0.0.1:22/opt/selenium-server-standalone.jar",
			},
			[]byte("test"),
			url.NewResource("scp://127.0.0.1:22/", credentialFile),
			&runner.StartRequest{
				Target:     target,
				Sdk:        "jdk",
				SdkVersion: "1.8",
				Version:    "3.4",
				Port:       8117,
			},
			28866,
		},
	}

	for _, useCase := range useCases {
		execService, err := exec.GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := exec.OpenTestContext(manager, useCase.target, execService)
			var state = context.State()

			if len(useCase.DataURLs) > 0 {
				storageService := storage.NewMemoryService()
				state.Put(tstorage.UseMemoryService, true)
				for _, setupURL := range useCase.DataURLs {
					err = storageService.Upload(setupURL, bytes.NewReader(useCase.DataPayload))
				}

				assert.Nil(t, err)
			}
			service, err := context.Service(runner.ServiceID)
			if !assert.Nil(t, err) {
				break
			}

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, useCase.request)

				var baseCase = useCase.baseDir
				assert.Equal(t, "", serviceResponse.Error, baseCase)
				response, ok := serviceResponse.Response.(*runner.StartResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", baseCase, serviceResponse.Response))
					continue
				}
				if response != nil {
					var actual = response.Pid
					assert.Equal(t, actual, useCase.Pid, "PID "+baseCase)
				}
			}
		}
	}

}

func StartSeleniumMockServer(port int) error {
	baseDir := toolbox.CallerDirectory(3)
	var sessionPath = path.Join(baseDir, "test/http/")
	return endpoint.StartServer(port, &endpoint.HTTPServerTrips{
		IndexKeys:     []string{endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey, endpoint.ContentTypeKey},
		BaseDirectory: sessionPath,
	})
}

func TestSeleniumService_Calls(t *testing.T) {

	StartSeleniumMockServer(8116)

	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	var targetHost = "127.0.0.1:8116"
	var target = url.NewResource(fmt.Sprintf("http://%v/", targetHost))

	serviceResponse := service.Run(context, &runner.OpenSessionRequest{
		RemoteSelenium: target,
		Browser:        "firefox",
	})

	if assert.Equal(t, "", serviceResponse.Error) {
		response, ok := serviceResponse.Response.(*runner.OpenSessionResponse)
		if assert.True(t, ok) {
			assert.EqualValues(t, response.SessionID, targetHost)
		}
	} else {
		return
	}

	serviceResponse = service.Run(context, &runner.WebDriverCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Get",
			Parameters: []interface{}{"http://play.golang.org/?simple=1"},
		},
	})

	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*runner.ServiceCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,

		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#dummay",
		},
		Call: &runner.MethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	})
	response, ok := serviceResponse.Response.(*runner.WebElementCallResponse)
	if assert.True(t, ok) {
		assert.Equal(t, "failed to lookup element: css selector #dummay", response.LookupError)
	}
	serviceResponse = service.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,

		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#code",
		},
		Call: &runner.MethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	})

	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*runner.WebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,

		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#code",
		},
		Call: &runner.MethodCall{
			Method: "SendKeys",
			Parameters: []interface{}{
				code,
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*runner.WebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Click",
			Parameters: []interface{}{},
			Wait:       &endly.Repeater{SleepTimeMs: 1},
		},
		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#run",
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*runner.WebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,

		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#output",
		},
		Call: &runner.MethodCall{
			Method:     "Text",
			Parameters: []interface{}{},
			Wait: &endly.Repeater{
				Repeat:      20,
				SleepTimeMs: 100,
				Exit:        "$value:/WebDriver/",
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		callResponse, ok := serviceResponse.Response.(*runner.WebElementCallResponse)
		if assert.True(t, ok) {
			assert.True(t, strings.Contains(toolbox.AsString(callResponse.Result[0]), "Hello WebDriver!"))
		}
	}

	serviceResponse = service.Run(context, &runner.WebDriverCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Close",
			Parameters: []interface{}{},
		},
	})

}

func TestSeleniumService_Run(t *testing.T) {

	StartSeleniumMockServer(8118)

	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	var targetHost = "127.0.0.1:8118"
	var target = url.NewResource(fmt.Sprintf("http://%v/", targetHost))

	serviceResponse := service.Run(context, &runner.RunRequest{
		RemoteSelenium: target,
		Browser:        "firefox",
		Actions: []*runner.Action{
			{
				Calls: []*runner.MethodCall{
					runner.NewMethodCall("Get", nil, "http://play.golang.org/?simple=1"),
				},
			},
			{
				Selector: runner.NewWebElementSelector("", "#code"),
				Calls: []*runner.MethodCall{
					runner.NewMethodCall("Clear", nil),
				},
			},
			{
				Selector: runner.NewWebElementSelector("", "#code"),
				Calls: []*runner.MethodCall{
					runner.NewMethodCall("SendKeys", nil, code),
				},
			},
			{
				Selector: runner.NewWebElementSelector("", "#run"),
				Calls: []*runner.MethodCall{
					runner.NewMethodCall("Click", nil),
				},
			},
			{
				Selector: runner.NewWebElementSelector("", "#output"),
				Calls: []*runner.MethodCall{
					runner.NewMethodCall("Text", &endly.Repeater{
						Repeat:      20,
						SleepTimeMs: 100,
						Exit:        "$value:/WebDriver/",
					}),
				},
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {

		runResponse, ok := serviceResponse.Response.(*runner.RunResponse)
		if assert.True(t, ok) {
			output, ok := runResponse.Data["#output"]
			if assert.True(t, ok) {
				outputMap := toolbox.AsMap(output)
				assert.EqualValues(t, "Hello WebDriver!\n\n\nProgram exited.", outputMap["Text"])
			}

		}
	}

	serviceResponse = service.Run(context, &runner.CloseSessionRequest{
		SessionID: targetHost,
	})

}
