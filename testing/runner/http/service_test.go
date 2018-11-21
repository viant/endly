package http_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/test/proto"
	endpoint "github.com/viant/endly/testing/endpoint/http"
	runner "github.com/viant/endly/testing/runner/http"
	"github.com/viant/endly/udf"
	_ "github.com/viant/endly/udf"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"log"
	"net/http"
	"path"
	"strings"
	"testing"
)

func StartTestServer(port int, basedir string, indexBy ...string) error {
	if len(indexBy) == 0 {
		indexBy = []string{endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey, endpoint.CookieKey, endpoint.ContentTypeKey}
	}
	baseDir := toolbox.CallerDirectory(3)
	return endpoint.StartServer(port, &endpoint.HTTPServerTrips{
		IndexKeys:     indexBy,
		BaseDirectory: path.Join(baseDir, basedir),
	})
}

func TestHttpRunnerService_Run(t *testing.T) {
	err := StartTestServer(8766, "test/send")
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &runner.SendRequest{
		Options: []*toolbox.HttpOptions{
			{
				Key:   "RequestTimeoutMs",
				Value: 12000,
			},
		},
		Requests: []*runner.Request{
			{
				URL:    "http://127.0.0.1:8766/send1",
				Method: "POST",
				Body:   "0123456789",
				Header: http.Header{
					"User-Agent": []string{"myUa"},
				},

				Repeater: &model.Repeater{
					Extraction: []*model.Extract{
						{
							Key:     "send_arg1",
							RegExpr: "send1 (.+)",
						},
					},
				},
			},
			{
				URL:    "http://127.0.0.1:8766/send1",
				Method: "POST",
				Body:   "0123456789",
				Repeater: &model.Repeater{
					Extraction: []*model.Extract{
						{
							Key:     "send_arg2",
							RegExpr: "send1 (.+)",
						},
					},
				},
			},
			{
				When:   "content1-2",
				URL:    "http://127.0.0.1:8766/send2",
				Method: "POST",
				Body:   "xc",
			},
		},
	})
	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*runner.SendResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 3, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-1", sendResponse.Data["send_arg1"])
	assert.Equal(t, "content1-2", sendResponse.Data["send_arg2"])

}

func TestHttpRunnerService_Repeat(t *testing.T) {
	err := StartTestServer(8111, "test/send")
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &runner.SendRequest{
		Requests: []*runner.Request{
			{
				URL:    "http://127.0.0.1:8111/send1",
				Method: "POST",
				Body:   "0123456789",
				Repeater: &model.Repeater{
					Extraction: []*model.Extract{
						{
							Key:     "send_arg1",
							RegExpr: "send1 (.+)",
						},
						{
							Key:     "send_arg2",
							RegExpr: "send1 (.+)",
						},
					},
					Repeat: 2,
				},
			},

			{
				When:   "${httpTrips.Response[0].Body}:/never-match/",
				URL:    "http://127.0.0.1:8111/send2",
				Method: "POST",
				Body:   "xc",
			},
			{
				When:   "${httpTrips.Response[0].Body}:/content1-2/",
				URL:    "http://127.0.0.1:8111/send2",
				Method: "POST",
				Body:   "xc",
			},
		},
	})

	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*runner.SendResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 2, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-2", sendResponse.Data["send_arg1"])
	assert.Equal(t, "content1-2", sendResponse.Data["send_arg2"])

}

func TestHttpRunnerService_RepeatWthExitCriteria(t *testing.T) {
	err := StartTestServer(8112, "test/send")
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &runner.SendRequest{
		Requests: []*runner.Request{
			{
				URL:    "http://127.0.0.1:8112/send1",
				Method: "POST",
				Body:   "0123456789",
				Repeater: &model.Repeater{
					Extraction: []*model.Extract{
						{
							Key:     "var1",
							RegExpr: "send1 (.+)",
						},
					},
					Repeat:      1000,
					SleepTimeMs: 500,
					Exit:        "$var1:content1-2",
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*runner.SendResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 1, len(sendResponse.Responses))
	for _, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code)
	}
	assert.Equal(t, "content1-2", sendResponse.Data["var1"])

}

func AsTestMessage(source interface{}, state data.Map) (interface{}, error) {
	return udf.AsProtobufMessage(source, state, &proto.Message{})
}

func FromTestMessage(source interface{}, state data.Map) (interface{}, error) {
	return udf.FromProtobufMessage(source, state, &proto.Message{})
}

func init() {
	endly.UdfRegistry["AsTestMessage"] = AsTestMessage
	endly.UdfRegistry["FromTestMessage"] = FromTestMessage
}

func TestHttpRunnerService_PayloadTransformation(t *testing.T) {
	err := StartTestServer(8119, "test/udf")
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())

	{ // valid udf use case
		response := service.Run(context, &runner.SendRequest{
			Requests: []*runner.Request{
				{
					URL:         "http://127.0.0.1:8119/udf1",
					Method:      "POST",
					Body:        "{\"id\":110, \"name\":\"test 1\"}",
					RequestUdf:  "AsTestMessage",
					ResponseUdf: "FromTestMessage",
				},
				{
					URL:         "http://127.0.0.1:8119/udf2",
					Method:      "POST",
					Body:        "{\"id\":121, \"name\":\"test 2\"}",
					RequestUdf:  "AsTestMessage",
					ResponseUdf: "FromTestMessage",
				},
			},
		})

		if assert.Equal(t, "", response.Error) {
			sendResponse, ok := response.Response.(*runner.SendResponse)
			if assert.True(t, ok) {
				assert.EqualValues(t, 2, len(sendResponse.Responses))
				for _, response := range sendResponse.Responses {
					assert.EqualValues(t, 200, response.Code)
					assert.EqualValues(t, 1, response.JSONBody["Id"])
					assert.EqualValues(t, "abc", response.JSONBody["Name"])
				}
			}
		}

	}

	{ // invalid request UDFs use case
		response := service.Run(context, &runner.SendRequest{
			Requests: []*runner.Request{
				{
					URL:         "http://127.0.0.1:8119/udf1",
					Method:      "POST",
					Body:        "{\"id\":110, \"name\":\"test 1\"}",
					RequestUdf:  "AsTestBlah",
					ResponseUdf: "FromTestMessage",
				},
			},
		})
		assert.True(t, strings.Contains(response.Error, "failed to lookup udf: AsTestBlah"))
	}

	{ // invalid json for request UDFs use case
		response := service.Run(context, &runner.SendRequest{
			Requests: []*runner.Request{
				{
					URL:         "http://127.0.0.1:8119/udf1",
					Method:      "POST",
					Body:        "{\"id:110, \"name\":\"test 1\"}",
					RequestUdf:  "AsTestMessage",
					ResponseUdf: "FromTestMessage",
				},
			},
		})
		assert.True(t, strings.Contains(response.Error, "failed to run udf"), response.Error)
	}

	{ // invalid response UDFs use case
		response := service.Run(context, &runner.SendRequest{
			Requests: []*runner.Request{
				{
					URL:         "http://127.0.0.1:8119/udf1",
					Method:      "POST",
					Body:        "{\"id\":110, \"name\":\"test 1\"}",
					RequestUdf:  "AsTestMessage",
					ResponseUdf: "FromTewewewswwtMessage",
				},
			},
		})
		assert.EqualValues(t, true, strings.Contains(response.Error, "failed to lookup udf: FromTewewewswwtMessage"))
	}

	{ //base 64 use case
		response := service.Run(context, &runner.SendRequest{
			Requests: []*runner.Request{
				{
					URL:    "http://127.0.0.1:8119/udf3",
					Method: "POST",
					Body:   "base64:CHkSBnRlc3QgMg==",
				},
			},
		})

		if assert.Equal(t, "", response.Error) {
			sendResponse, ok := response.Response.(*runner.SendResponse)
			if assert.True(t, ok) {
				assert.EqualValues(t, 1, len(sendResponse.Responses))
				assert.EqualValues(t, "base64:CAESA2FiYw==", sendResponse.Responses[0].Body)
			}
		}
	}
}

func Test_UdfProvider(t *testing.T) {
	var parentDir = toolbox.CallerDirectory(3)
	err := StartTestServer(8987, "test/udf_provider", endpoint.MethodKey, endpoint.URLKey)
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	registerUDF, err := udf.NewRegisterRequestFromURL(path.Join(parentDir, "test/register_udf.json"))
	if !assert.Nil(t, err) {
		return
	}
	err = endly.Run(context, registerUDF, nil)
	if !assert.Nil(t, err) {
		return
	}
	request, err := runner.NewSendRequestFromURL(path.Join(parentDir, "test/udf_provider.json"))
	if !assert.Nil(t, err) {
		return
	}
	var response = &runner.SendResponse{}
	err = endly.Run(context, request, response)
	if !assert.Nil(t, err) {
		log.Fatal(err)
		return
	}
	var expected = `{
	"Data": {
		"value": "{\"Desc\":\"abc\",\"ID\":1}\n"
	},
	"Responses": [
		{
			"Body": "{\"Desc\":\"abc\",\"ID\":1}\n",
			"Code": 200,
			"Header": {
				"Content-Length": [
					"183"
				],
				"Content-Type": [
					"application/avro-binary; Charset=UTF-8"
				]
			},
			"JSONBody": {
				"Desc": "abc",
				"ID": 1
			}
		}
	]
}
`
	assertly.AssertValues(t, expected, response)

}
