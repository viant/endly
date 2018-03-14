package http_test

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	endpoint "github.com/viant/endly/endpoint/http"
	runner "github.com/viant/endly/runner/http"
	"github.com/viant/endly/test/proto"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

func StartTestServer(port int, basedir string) error {
	baseDir := toolbox.CallerDirectory(3)
	return endpoint.StartServer(port, &endpoint.HTTPServerTrips{
		IndexKeys:     []string{endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey, endpoint.CookieKey, endpoint.ContentTypeKey},
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
		Requests: []*runner.Request{
			{
				URL:    "http://127.0.0.1:8766/send1",
				Method: "POST",
				Body:   "0123456789",
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
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
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
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


// Test for checking "When" criteria with request/response dependency
func TestHttpRunnerService_Run_Expansion(t *testing.T) {
	manager := endly.New()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &runner.SendRequest{
		Requests: []*runner.Request{
			{
				URL:    "http://127.0.0.1:9880/d/rtbv2/rubicon/bidder",
				Method: "POST",
				Body:   "{\"at\":3,\"cur\":[\"USD\"],\"device\":{\"devicetype\":2,\"geo\":{\"country\":\"usa\",\"lat\":0,\"lon\":0},\"ip\":\"207.255.87.114\",\"language\":\"en\",\"ua\":\"Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/38.0.2125.19 Safari/537.36\"},\"ext\":{\"adtricity\":{\"score\":1.8497373,\"viewability\":56.984},\"xid\":\"2820387111204119977\"},\"id\":\"5d9bfb65-cf1c-2753-a392-42010af177d7\",\"imp\":[{\"bidfloor\":0,\"bidfloorcur\":\"USD\",\"id\":\"1\",\"pmp\":{\"deals\":[{\"at\":3,\"bidfloor\":1,\"bidfloorcur\":\"USD\",\"ext\":{\"dealpoints\":\"OPTION_BID\"},\"id\":\"rubicon_high_forecast\"}],\"private_auction\":1},\"secure\":1,\"video\":{\"api\":[2],\"battr\":[],\"boxingallowed\":1,\"delivery\":[1,2],\"h\":250,\"linearity\":1,\"maxbitrate\":0,\"maxduration\":15,\"mimes\":[\"video/webm\",\"video/mp4\",\"video/x-flv\",\"application/javascript\"],\"minbitrate\":300,\"minduration\":1,\"playbackmethod\":[1],\"protocols\":[5,2,3],\"startdelay\":0,\"w\":300}}],\"site\":{\"cat\":[\"IAB24\"],\"domain\":\"popularmechanics.com\",\"id\":\"554257\",\"name\":\"popularmechanics.com\",\"page\":\"http://www.popularmechanics.com/science/animals/a13205/this-is-what-an-electric-eel-shocks-does-17493859/\",\"ref\":\"http://www.popularmechanics.com/science/animals/a13205/this-is-what-an-electric-eel-shocks-does-17493859/\"},\"ts\":\"1520538483963\",\"user\":{\"id\":\"74e24e93-b7a0-408a-8c29-e18f823ceac6\"}}",
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
						{
							Key:     "winNoticePath",
							RegExpr: "(/rubicon/won[^\"]+)",
							Reset:   true,
						},
					},
					Variables: endly.Variables{
						{
							Name:     "AUCTION_ID",
							From:     "id",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "AUCTION_BID_ID",
							From:     "seatbid[0].bid[0].id",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "AUCTION_IMP_ID",
							From:     "seatbid[0].bid[0].impid",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "AUCTION_SEAT_ID",
							From:     "seatbid[0].seat",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "AUCTION_CURRENCY",
							From:     "cur",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "AUCTION_PRICE:BF",
							Value:    "54F0C3E48BF91D38E6FDE072443AF70A",
							Persist:  false,
							Required: false,
						},
							{
							Name:     "winNoticeURL",
							Value:    "http://127.0.0.1:8990/logger/rtb${winNoticePath}",
							From:     "",
							Persist:  false,
							Required: false,
						},
					},
				},
			},
			{
				When: "${responses[0].Body:/rubicon/won}",
				URL:       "${previous.winNoticeURL}",
				Method:    "GET",
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
						{
							Key:     "csImpPath",
							RegExpr: "(/csimp/pixel[^\\]]+)",
							Reset:   true,
						},
						{
							Key:     "secondWinPath",
							RegExpr: "(/rubicon/won[^\\]]+)",
							Reset:   true,
						},
					},
					Variables: endly.Variables{
						{
							Name:     "csImpURL",
							Value:    "http://127.0.0.1:8990/logger${csImpPath}",
							Persist:  false,
							Required: false,
						},
						{
							Name:     "secondWinURL",
							Value:    "http://127.0.0.1:8990/logger${secondWinPath}",
							Persist:  false,
							Required: false,
						},
					},
				},
			},
			{
				When: "csimp/pixel",
				URL:       "${previous.csImpURL}",
				Method:    "GET",
			},
		},
	})
	assert.Equal(t, "", response.Error)
	sendResponse, ok := response.Response.(*runner.SendResponse)
	assert.True(t, ok)
	assert.EqualValues(t, 3, len(sendResponse.Responses))
	for i, response := range sendResponse.Responses {
		assert.EqualValues(t, 200, response.Code, fmt.Sprintf("Incorrect code from response number:%d", i))
	}

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
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
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
				When:   "never-match",
				URL:    "http://127.0.0.1:8111/send2",
				Method: "POST",
				Body:   "xc",
			},
			{
				When:   "content1-2",
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
				Repeater: &endly.Repeater{
					Extraction: []*endly.Extract{
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
	return endly.AsProtobufMessage(source, state, &proto.Message{})
}

func FromTestMessage(source interface{}, state data.Map) (interface{}, error) {
	return endly.FromProtobufMessage(source, state, &proto.Message{})
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

	{ // invalid request UDF use case
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

	{ // invalid json for request UDF use case
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

	{ // invalid response UDF use case
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
