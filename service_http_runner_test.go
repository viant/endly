package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"path"
	"github.com/viant/toolbox/data"
	"github.com/viant/endly/test/proto"
	"strings"
)

func StartTestServer(port int, basedir string) error {
	baseDir := toolbox.CallerDirectory(3)
	return endly.StartHttpServer(port, &endly.HttpServerTrips{
		IndexKeys:[]string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
		BaseDirectory:path.Join(baseDir, basedir),
	})
}



func TestHttpRunnerService_Run(t *testing.T) {
	err := StartTestServer(8766, "test/http/runner/send")
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
	err := StartTestServer(8111, "test/http/runner/send")
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
	err := StartTestServer(8112, "test/http/runner/send")
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


func AsTestMessage(source interface{}, state data.Map) (interface{}, error) {
	return endly.AsProtobufMessage(source, state, &proto.Message{})
}

func FromTestMessage(source interface{}, state data.Map) (interface{}, error) {
	return endly.FromProtobufMessage(source, state, &proto.Message{})
}

func init() {
	endly.UdfRegistry["AsTestMessage"] = AsTestMessage
	endly.UdfRegistry["FromTestMessage"] = FromTestMessage
}






func TestHttpRunnerService_RunUdf(t *testing.T) {
	err := StartTestServer(8119, "test/http/runner/udf")
	if ! assert.Nil(t, err) {
		return
	}

	manager := endly.NewManager()
	service, err := manager.Service(endly.HTTPRunnerServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)


	context := manager.NewContext(toolbox.NewContext())

	response := service.Run(context, &endly.SendHTTPRequest{
		RequestUdf:"AsTestBlah",
		ResponseUdf:"AsTestBlah",
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8119/udf1",
				Method:"POST",
				Body:"{\"id\":110, \"name\":\"test 1\"}",

			},

		},
	})
	assert.True(t, strings.Contains(response.Error, "failed to lookup udf: AsTestBlah"))

	response = service.Run(context, &endly.SendHTTPRequest{
		RequestUdf:"AsTestMessage",
		ResponseUdf:"FromTestMessage",
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8119/udf1",
				Method:"POST",
				Body:"{\"id:110, \"name\":\"test 1\"}",
			},

		},
	})
	assert.True(t, strings.Contains(response.Error, "unable to run udf"), response.Error)


	response = service.Run(context, &endly.SendHTTPRequest{
		RequestUdf:"AsTestMessage",
		ResponseUdf:"FromTewewewswwtMessage",
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8119/udf1",
				Method:"POST",
				Body:"{\"id\":110, \"name\":\"test 1\"}",

			},

		},
	})
	assert.EqualValues(t, true, strings.Contains(response.Error, "failed to lookup udf: FromTewewewswwtMessage"))

	response = service.Run(context, &endly.SendHTTPRequest{
		RequestUdf:"AsTestMessage",
		ResponseUdf:"FromTestMessage",
		Requests: []*endly.HTTPRequest{
			{
				URL: "http://127.0.0.1:8119/udf1",
				Method:"POST",
				Body:"{\"id\":110, \"name\":\"test 1\"}",

			},
			{
				URL: "http://127.0.0.1:8119/udf2",
				Method:"POST",
				Body:"{\"id\":121, \"name\":\"test 2\"}",

			},
		},
	})

	if assert.Equal(t, "", response.Error) {
		sendResponse, ok := response.Response.(*endly.SendHTTPResponse)
		if assert.True(t, ok) {
			assert.EqualValues(t, 2, len(sendResponse.Responses))
			for _, response := range sendResponse.Responses {
				assert.EqualValues(t, 200, response.Code)
				assert.EqualValues(t, 1 ,response.JSONBody["Id"])
				assert.EqualValues(t, "abc" ,response.JSONBody["Name"])
			}
		}
	}


}


//TODO add udf, base64 use cases