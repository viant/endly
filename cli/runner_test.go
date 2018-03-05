package cli_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/cli"
	"github.com/viant/toolbox"
	"log"
	"os/exec"
	"path"
	"testing"
)

func TestCliRunner_RunDsUnitWorkflow(t *testing.T) {
	exec.Command("rm", "-rf", "/tmp/endly/test/workflow/dsunit").CombinedOutput()
	toolbox.CreateDirIfNotExist("/tmp/endly/test/workflow/dsunit")
	runner := cli.New()

	request, options, err := cli.LoadRunRequestWithOption("test/runner/run_dsunit.json")
	if assert.Nil(t, err) {
		err := runner.Run(request, options)
		if err != nil {
			log.Fatal(err)
		}
	}
}

//
//func TestCliRunner_RunDsHttpWorkflow(t *testing.T) {
//	baseDir := toolbox.CallerDirectory(3)
//	err := endly.StartHTTPServer(8120, &endly.HTTPServerTrips{
//		IndexKeys:     []string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
//		BaseDirectory: path.Join(baseDir, "test/http/runner/http_workflow"),
//	})
//
//	if !assert.Nil(t, err) {
//		return
//	}
//	toolbox.CreateDirIfNotExist("/tmp/endly/test/workflow/dsunit")
//	runner := cli.New()
//
//	origin := endly.OnError
//	defer func() {
//		endly.OnError = origin
//	}()
//	endly.OnError = func(code int) {
//
//	}
//
//	request, options, err := endly.LoadRunRequestWithOption("test/runner/run_http.json")
//	if assert.Nil(t, err) {
//		err := runner.Run(request, options)
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
//}
//
//func Test_LoadRunRequestWithOption(t *testing.T) {
//
//	{ //non existing file
//		_, _, err := endly.LoadRunRequestWithOption("test/runner/abc.json")
//		assert.NotNil(t, err)
//	}
//
//	{ //inalid JSON
//		_, _, err := endly.LoadRunRequestWithOption("test/runner/run_malformed.json")
//		assert.NotNil(t, err)
//	}
//
//}
//
//func Test_DefaultRunnerReportingOption(t *testing.T) {
//	options := endly.DefaultRunnerReportingOption()
//	assert.NotNil(t, options)
//}
//
//func TestCliRunner_Run(t *testing.T) {
//	runner := endly.NewCliRunner()
//	endly.OnError = func(code int) {
//
//	}
//	{
//		err := runner.Run(&endly.WorkflowRunRequest{
//			WorkflowURL: "action",
//			Tasks:       "run",
//			Params: map[string]interface{}{
//				"service": "logger",
//				"action":  "print",
//				"request": &endly.LoggerPrintRequest{StyledText: "hello"},
//			},
//		}, nil)
//
//		assert.Nil(t, err)
//	}
//	{
//
//		err := runner.Run(&endly.WorkflowRunRequest{
//			WorkflowURL: "action",
//			Tasks:       "run",
//			Params: map[string]interface{}{
//				"service": "workflow",
//				"action":  "fail",
//				"request": &endly.WorkflowFailRequest{StyledText: "hello"},
//			},
//		}, nil)
//		assert.NotNil(t, err)
//	}
//}
//
//func Test_Run(t *testing.T) {
//
//}
