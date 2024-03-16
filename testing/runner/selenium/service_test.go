package selenium_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/location"
	_ "github.com/viant/endly/shared"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	endpoint "github.com/viant/endly/testing/endpoint/http"
	runner "github.com/viant/endly/testing/runner/selenium"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
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
		target      *location.Resource
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
				Port:       5617,
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
				Port:       5617,
			},
			28866,
		},
	}

	for _, useCase := range useCases {
		if assert.Nil(t, err) {
			context, err := exec.NewSSHReplayContext(manager, useCase.target, useCase.baseDir)
			if !assert.Nil(t, err) {
				continue
			}
			if len(useCase.DataURLs) > 0 {
				storageService := storage.UseMemoryService(context)
				for _, setupURL := range useCase.DataURLs {
					storageService.Upload(setupURL, bytes.NewReader(useCase.DataPayload))
				}
			}
			var description = useCase.baseDir
			var response = &runner.StartResponse{}
			err = endly.Run(context, useCase.request, response)
			if !assert.Nil(t, err, description) {
				t.Error(err.Error())
				continue
			}

			var actual = response.Pid
			assert.Equal(t, actual, useCase.Pid, "PID "+description)
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

	StartSeleniumMockServer(5619)

	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	var targetHost = "127.0.0.1:5619"
	var target = url.NewResource(fmt.Sprintf("http://%v/", targetHost))

	var openResponse = &exec.OpenSessionResponse{}
	if err := endly.Run(context, &runner.OpenSessionRequest{
		RemoteSelenium: target,
		Browser:        "firefox",
	}, openResponse); err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, openResponse.SessionID, targetHost)

	if err = endly.Run(context, &runner.WebDriverCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Get",
			Parameters: []interface{}{"http://play.golang.org/?simple=1"},
		},
	}, nil); err != nil {
		t.Fatal(err)
	}

	var response = &runner.WebElementCallResponse{}
	if err = endly.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,
		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#dummay",
		},
		Call: &runner.MethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	}, response); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "failed to lookup element: css selector #dummay", response.LookupError)

	if err = endly.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,
		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#code",
		},
		Call: &runner.MethodCall{
			Method:     "Clear",
			Parameters: []interface{}{},
		},
	}, nil); err != nil {
		t.Fatal(err)
	}

	if err = endly.Run(context, &runner.WebElementCallRequest{
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
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err = endly.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Click",
			Parameters: []interface{}{},
			Wait:       &model.Repeater{SleepTimeMs: 1},
		},
		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#run",
		},
	}, nil); err != nil {
		t.Fatal(err)
	}

	if err = endly.Run(context, &runner.WebElementCallRequest{
		SessionID: targetHost,
		Selector: &runner.WebElementSelector{
			By:    "css selector",
			Value: "#output",
		},
		Call: &runner.MethodCall{
			Method:     "Text",
			Parameters: []interface{}{},
			Wait: &model.Repeater{
				Repeat:      20,
				SleepTimeMs: 100,
				Exit:        "$value:/WebDriver/",
			},
		},
	}, response); err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(toolbox.AsString(response.Result[0]), "Hello WebDriver!"))

	if err = endly.Run(context, &runner.WebDriverCallRequest{
		SessionID: targetHost,
		Call: &runner.MethodCall{
			Method:     "Close",
			Parameters: []interface{}{},
		},
	}, nil); err != nil {
		t.Fatal(err)
	}

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
					runner.NewMethodCall("Text", &model.Repeater{
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
