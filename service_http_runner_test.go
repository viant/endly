package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"path"
)

func StartTestServer(port int) error {
	baseDir := toolbox.CallerDirectory(3)
	return endly.StartHttpServer(port, &endly.HttpServerTrips{
		IndexKeys:[]string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
		BaseDirectory:path.Join(baseDir, "test/http/runner/send"),
	})
}



func TestHttpRunnerService_Run(t *testing.T) {
	err := StartTestServer(8766)
	if ! assert.Nil(t, err) {
		return
	}
	manager := endly.NewManager()
	service, err := manager.Service(endly.HTTPRunnerServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.SendHTTPRequest{
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8766/send1",
				Method:"POST",
				Body:"0123456789",
				Extraction: []*endly.DataExtraction{
					{
						Key:     "send_arg1",
						RegExpr: "send1 (.+)",
					},
				},
			},
			{
				URL: "http://127.0.0.1:8766/send1",
				Method:"POST",
				Body:"0123456789",
				Extraction: []*endly.DataExtraction{
					{
						Key:     "send_arg2",
						RegExpr: "send1 (.+)",
					},
				},
			},
			{
				MatchBody:"content1-2",
				URL: "http://127.0.0.1:8766/send2",
				Method:"POST",
				Body:"xc",

			},
		},
	})
	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*endly.SendHTTPResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 3, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-1", sendResponse.Extracted["send_arg1"])
	assert.Equal(t, "content1-2", sendResponse.Extracted["send_arg2"])


}




func TestHttpRunnerService_Repeat(t *testing.T) {
	err := StartTestServer(8111)
	if ! assert.Nil(t, err) {
		return
	}
	manager := endly.NewManager()
	service, err := manager.Service(endly.HTTPRunnerServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.SendHTTPRequest{
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8111/send1",
				Method:"POST",
				Body:"0123456789",
				Extraction: []*endly.DataExtraction{
					{
						Key:     "send_arg1",
						RegExpr: "send1 (.+)",
					},
					{
						Key:     "send_arg2",
						RegExpr: "send1 (.+)",
					},
				},
				Repeat:2,
			},

			{
				MatchBody:"never-match",
				URL: "http://127.0.0.1:8111/send2",
				Method:"POST",
				Body:"xc",

			},
			{
				MatchBody:"content1-2",
				URL: "http://127.0.0.1:8111/send2",
				Method:"POST",
				Body:"xc",

			},

		},
	})


	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*endly.SendHTTPResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 2, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-2", sendResponse.Extracted["send_arg1"])
	assert.Equal(t, "content1-2", sendResponse.Extracted["send_arg2"])


}



func TestHttpRunnerService_RepeatWthExitCriteria(t *testing.T) {
	err := StartTestServer(8112)
	if ! assert.Nil(t, err) {
		return
	}
	manager := endly.NewManager()
	service, err := manager.Service(endly.HTTPRunnerServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.SendHTTPRequest{
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8112/send1",
				Method:"POST",
				Body:"0123456789",
				Extraction: []*endly.DataExtraction{
					{
						Key:     "var1",
						RegExpr: "send1 (.+)",
					},
				},
				Repeat:1000,
				SleepTimeMs:500,
				ExitCriteria:"$var1:content1-2",
			},
		},
	})


	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*endly.SendHTTPResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 1, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-2", sendResponse.Extracted["var1"])


}


//TODO add udf, base64 use cases