package endly_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"

	"path"
)

const code = `
	package main
	import "fmt"

	func main() {
		fmt.Println("Hello WebDriver!\n")
	}
`

func TestSeleniumService_Start(t *testing.T) {

	var credentialFile, err = GetDummyCredential()
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile)
	assert.Nil(t, err)
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir     string
		DataURLs    []string
		DataPayload []byte
		target      *url.Resource
		request     *endly.SeleniumServerStartRequest
		Pid         int
	}{
		{
			"test/selenium/start/inactive/darwin",
			[]string{
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-linux64.tar.gz",
				"http://selenium-release.storage.googleapis.com/3.4/selenium-server-standalone-3.4.0.jar",
			},
			[]byte("test"),
			url.NewResource("scp://127.0.0.1:22/", credentialFile),
			&endly.SeleniumServerStartRequest{
				Target:     target,
				Sdk:        "jdk",
				SdkVersion: "1.8",
				Version:    "3.4",
				Port:       8117,
			},
			28811,
		},
		{
			"test/selenium/start/active/darwin",
			[]string{
				"https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-linux64.tar.gz",
				"http://selenium-release.storage.googleapis.com/3.4/selenium-server-standalone-3.4.0.jar",
			},
			[]byte("test"),
			url.NewResource("scp://127.0.0.1:22/", credentialFile),
			&endly.SeleniumServerStartRequest{
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
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, useCase.target, execService)
			var state = context.State()

			if len(useCase.DataURLs) > 0 {
				storageService := storage.NewMemoryService()
				state.Put(endly.UseMemoryService, true)
				for _, setupURL := range useCase.DataURLs {
					err = storageService.Upload(setupURL, bytes.NewReader(useCase.DataPayload))
				}

				assert.Nil(t, err)
			}
			service, err := context.Service(endly.SeleniumServiceID)
			if !assert.Nil(t, err) {
				break
			}

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, useCase.request)

				var baseCase = useCase.baseDir
				assert.Equal(t, "", serviceResponse.Error, baseCase)
				response, ok := serviceResponse.Response.(*endly.SeleniumServerStartResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", baseCase, serviceResponse.Response))
					continue
				}
				var actual = response.Pid
				assert.Equal(t, actual, useCase.Pid, "PID "+baseCase)
			}
		}
	}

}

func StartSeleniumMockServer(port int) error {
	baseDir := toolbox.CallerDirectory(3)
	var sessionPath = path.Join(baseDir, "test/selenium/http/")

	return endly.StartHTTPServer(port, &endly.HTTPServerTrips{
		IndexKeys:     []string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.ContentTypeKey},
		BaseDirectory: sessionPath,
	})
}

func TestSeleniumService_Calls(t *testing.T) {

	StartSeleniumMockServer(8116)

	manager := endly.NewManager()
	service, err := manager.Service(endly.SeleniumServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	var targetHost = "127.0.0.1:8116"
	var target = url.NewResource(fmt.Sprintf("http://%v/", targetHost))

	serviceResponse := service.Run(context, &endly.SeleniumOpenSessionRequest{
		RemoteSelenium: target,
		Browser:        "firefox",
	})

	if assert.Equal(t, "", serviceResponse.Error) {
		response, ok := serviceResponse.Response.(*endly.SeleniumOpenSessionResponse)
		if assert.True(t, ok) {
			assert.EqualValues(t, response.SessionID, targetHost)
		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebDriverCallRequest{
		SessionID: targetHost,
		Call: &endly.SeleniumMethodCall{
			Method:     "Get",
			Parameters: []interface{}{"http://play.golang.org/?simple=1"},
		},
	})

	if assert.Equal(t, "", serviceResponse.Error) {

		_, ok := serviceResponse.Response.(*endly.SeleniumServiceCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebElementCallRequest{
		SessionID: targetHost,

		Selector: &endly.WebElementSelector{
			By:    "css selector",
			Value: "#dummay",
		},
		Call: &endly.SeleniumMethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	})
	response, ok := serviceResponse.Response.(*endly.SeleniumWebElementCallResponse)
	if assert.True(t, ok) {
		assert.Equal(t, "failed to lookup element: css selector #dummay", response.LookupError)
	}
	serviceResponse = service.Run(context, &endly.SeleniumWebElementCallRequest{
		SessionID: targetHost,

		Selector: &endly.WebElementSelector{
			By:    "css selector",
			Value: "#code",
		},
		Call: &endly.SeleniumMethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	})

	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*endly.SeleniumWebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebElementCallRequest{
		SessionID: targetHost,

		Selector: &endly.WebElementSelector{
			By:    "css selector",
			Value: "#code",
		},
		Call: &endly.SeleniumMethodCall{
			Method: "SendKeys",
			Parameters: []interface{}{
				code,
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*endly.SeleniumWebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebElementCallRequest{
		SessionID: targetHost,
		Call: &endly.SeleniumMethodCall{
			Method:     "Click",
			Parameters: []interface{}{},
			Wait:       &endly.SeleniumWait{SleepInMs: 1},
		},
		Selector: &endly.WebElementSelector{
			By:    "css selector",
			Value: "#run",
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		_, ok := serviceResponse.Response.(*endly.SeleniumWebElementCallResponse)
		if assert.True(t, ok) {

		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebElementCallRequest{
		SessionID: targetHost,

		Selector: &endly.WebElementSelector{
			By:    "css selector",
			Value: "#output",
		},
		Call: &endly.SeleniumMethodCall{
			Method:     "Text",
			Parameters: []interface{}{},
			Wait: &endly.SeleniumWait{
				Repeat:       20,
				SleepInMs:    100,
				ExitCriteria: "$value:/WebDriver/",
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {
		callResponse, ok := serviceResponse.Response.(*endly.SeleniumWebElementCallResponse)
		if assert.True(t, ok) {
			assert.True(t, strings.Contains(toolbox.AsString(callResponse.Result[0]), "Hello WebDriver!"))
		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumWebDriverCallRequest{
		SessionID: targetHost,
		Call: &endly.SeleniumMethodCall{
			Method:     "Close",
			Parameters: []interface{}{},
		},
	})

}

func TestSeleniumService_Run(t *testing.T) {

	StartSeleniumMockServer(8118)

	manager := endly.NewManager()
	service, err := manager.Service(endly.SeleniumServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	var targetHost = "127.0.0.1:8118"
	var target = url.NewResource(fmt.Sprintf("http://%v/", targetHost))

	serviceResponse := service.Run(context, &endly.SeleniumRunRequest{
		RemoteSelenium: target,
		Browser:        "firefox",
		Actions: []*endly.SeleniumAction{
			{
				Calls: []*endly.SeleniumMethodCall{
					endly.NewSeleniumMethodCall("Get", nil, "http://play.golang.org/?simple=1"),
				},
			},
			{
				Selector: endly.NewWebElementSelector("", "#code"),
				Calls: []*endly.SeleniumMethodCall{
					endly.NewSeleniumMethodCall("Clear", nil),
				},
			},
			{
				Selector: endly.NewWebElementSelector("", "#code"),
				Calls: []*endly.SeleniumMethodCall{
					endly.NewSeleniumMethodCall("SendKeys", nil, code),
				},
			},
			{
				Selector: endly.NewWebElementSelector("", "#run"),
				Calls: []*endly.SeleniumMethodCall{
					endly.NewSeleniumMethodCall("Click", nil),
				},
			},
			{
				Selector: endly.NewWebElementSelector("", "#output"),
				Calls: []*endly.SeleniumMethodCall{
					endly.NewSeleniumMethodCall("Text", &endly.SeleniumWait{
						Repeat:       20,
						SleepInMs:    100,
						ExitCriteria: "$value:/WebDriver/",
					}),
				},
			},
		},
	})
	if assert.Equal(t, "", serviceResponse.Error) {

		runResponse, ok := serviceResponse.Response.(*endly.SeleniumRunResponse)
		if assert.True(t, ok) {
			output, ok := runResponse.Data["#output"]
			if assert.True(t, ok) {
				ouputMap := toolbox.AsMap(output)
				assert.EqualValues(t, "Hello WebDriver!\n\n\nProgram exited.", ouputMap["Text"])
			}

		}
	}

	serviceResponse = service.Run(context, &endly.SeleniumCloseSessionRequest{
		SessionID: targetHost,
	})

}
