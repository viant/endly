package http_test

import (
	"fmt"
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

func StartTestServer(port int, basedir string, rotate bool, indexBy ...string) error {
	if len(indexBy) == 0 {
		indexBy = []string{endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey, endpoint.CookieKey, endpoint.ContentTypeKey}
	}
	baseDir := toolbox.CallerDirectory(3)
	return endpoint.StartServer(port, &endpoint.HTTPServerTrips{
		IndexKeys:     indexBy,
		Rotate:        rotate,
		BaseDirectory: path.Join(baseDir, basedir),
	})
}

func TestHttpRunnerService_Run(t *testing.T) {
	err := StartTestServer(8766, "test/send", false)
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &runner.SendRequest{
		Options: map[string]interface{}{
			"RequestTimeoutMs": 12000,
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
					Extract: []*model.Extract{
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
					Extract: []*model.Extract{
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
	err := StartTestServer(8111, "test/send", false)
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
					Extract: []*model.Extract{
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
	err := StartTestServer(8112, "test/send", false)
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
					Extract: []*model.Extract{
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
	err := StartTestServer(8119, "test/udf", false)
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
	err := StartTestServer(8987, "test/udf_provider", false, endpoint.MethodKey, endpoint.URLKey)
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

func TestHttpRunnerService_Run_StressTest(t *testing.T) {
	err := StartTestServer(8988, "test/stress", true, endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey)
	if !assert.Nil(t, err) {
		log.Fatal(err)
	}
	request := &runner.LoadRequest{
		SendRequest: &runner.SendRequest{},
		ThreadCount: 6,
		Repeat:      30,
	}

	request.Requests = []*runner.Request{}
	var expected = []interface{}{}
	for i := 0; i < 6; i++ {
		request.Requests = append(request.Requests,
			&runner.Request{
				Method: "POST",
				URL:    fmt.Sprintf("http://127.0.0.1:8988/send%d", i),
				Body:   strings.Repeat(toolbox.AsString(i), 3),
			},
		)
		expected = append(expected,
			map[string]interface{}{
				"Code": 200,
				"Body": "1" + strings.Repeat(toolbox.AsString(i), 3),
			})
	}

	request.Expect = map[string]interface{}{
		"Responses": expected,
	}

	response := &runner.LoadResponse{}
	err = endly.Run(nil, request, response)
	if !assert.Nil(t, err) {
		log.Fatal(err)
	}
	expect := `{
	"RequestCount": 180,
	"Responses": [
		{
			"Body": "1000",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		},
		{
			"Body": "1111",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		},
		{
			"Body": "1222",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		},
		{
			"Body": "3333",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		},
		{
			"Body": "1444",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		},
		{
			"Body": "1555",
			"Code": 200,
			"Header": {
				"Cache-Control": [
					"private"
				],
				"Content-Length": [
					"4"
				],
				"Content-Type": [
					"text/html;charset=ISO-8859-1"
				],
				"Server": [
					"Apache-Coyote/1.1"
				]
			}
		}
	],
	"Status": "ok"
}`
	assertly.AssertValues(t, expect, response)

}

func TestRequest_FROMYaml(t *testing.T) {
	var JSON = `{
	"requests": [
		[
			{
				"Key": "URL",
				"Value": "http://localhost:5000"
			},
			{
				"Key": "Method",
				"Value": "GET"
			},
			{
				"Key": "Header",
				"Value": [
					{
						"Key": "aHeader",
						"Value": [
							"myField=a-value; path=/; domain=localhost; Expires=Tue, 19 Jan 2038 03:14:07 GMT;"
						]
					}
				]
			},
			{
				"Key": "Body",
				"Value": "hey there"
			},
			{
				"Key": "Cookies",
				"Value": [
					[
						{
							"Key": "Name",
							"Value": "aHeader"
						},
						{
							"Key": "Value",
							"Value": "a-value"
						},
						{
							"Key": "Domain",
							"Value": "localhost"
						},
						{
							"Key": "Expires",
							"Value": "2023-12-16T20:17:38Z"
						},
						{
							"Key": "RawExpires",
							"Value": "Sat, 16 Dec 2023 20:17:38 GMT"
						}
					]
				]
			}
		]
	]
}`

	var aMap = make(map[string]interface{})
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(JSON)).Decode(&aMap)
	if !assert.Nil(t, err) {
		log.Fatal(err)
	}
	var request = &runner.SendRequest{}
	err = toolbox.DefaultConverter.AssignConverted(request, aMap)
	if !assert.Nil(t, err) {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", request)

}
