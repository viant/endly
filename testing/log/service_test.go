package log_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/testing/log"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"
)

/*
	TODO rewrite all these test - this is mess
*/

var templateLog = map[string]interface{}{
	"k1": "v1",
	"k2": []string{"1", "2", "%v"},
	"k3": 123,
	"k4": map[string]interface{}{
		"s1": 1,
		"s2": "%v",
	},
	"k5": "— -%v",
}

func BuildLogContent(from, to, multiplier int, template string) string {
	var result = make([]string, 0)
	for i := from; i <= to; i++ {
		result = append(result, fmt.Sprintf(template, multiplier*i, (100*1)+1, 10*i))
	}
	return strings.Join(result, "")
}

func TestLogValidatorService_NewRequest(t *testing.T) {
	manager := endly.New()
	service, err := manager.Service(log.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()
	tempPath := path.Join(os.TempDir(), toolbox.AsString(time.Now().Unix()))
	err = os.Mkdir(tempPath, 0755)
	assert.Nil(t, err)
	var template, _ = toolbox.AsJSONText(templateLog)

	var fileURL = strings.Replace(location.NewResource(tempPath).URL, "file://", "scp://127.0.0.1", 1)
	credentials, err := util.GetDummyCredential()
	assert.Nil(t, err)
	var response = service.Run(context, &log.ListenRequest{
		Source: location.NewResource(fileURL, credentials),
		Types: []*log.Type{
			{
				Name:   "t",
				Format: "json",
				Mask:   "*.log",
			},
		},
	})

	for i := 0; i < 2; i++ {
		var logName = fmt.Sprintf("test%v.log", i)
		var fullLogname = path.Join(tempPath, logName)

		_ = toolbox.RemoveFileIfExist(fullLogname)
		var logContent = BuildLogContent(1, 3, i+1, template)
		err = ioutil.WriteFile(fullLogname, []byte(logContent), 0644)
		if err != nil {
			assert.FailNow(t, fmt.Sprintf("%v", err))
		}
		time.Sleep(time.Second)
	}

	assert.Equal(t, "", response.Error)
	var listenResponse, ok = response.Response.(*log.ListenResponse)
	if !assert.True(t, ok) {
		return
	}
	if !assert.NotNil(t, listenResponse) {
		return
	}

	logTypeMeta, ok := listenResponse.Meta["t"]
	assert.True(t, ok)
	assert.NotNil(t, logTypeMeta)
	assert.True(t, strings.HasSuffix(logTypeMeta.Source.URL, tempPath))
	assert.True(t, len(logTypeMeta.LogFiles) >= 1)

	response = service.Run(context, &log.AssertRequest{
		LogWaitTimeMs:     3000,
		LogWaitRetryCount: 3,
		Expect: []*log.TypedRecord{

			{
				Type: "t",
				Records: []interface{}{
					map[string]interface{}{
						"k5": "— -10",
					},
					map[string]interface{}{
						"k5": "— -20",
					},
					map[string]interface{}{
						"k5": "— -30",
					},
					map[string]interface{}{
						"k5": "— -10",
					},
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)
	logValidatorAssertResponse, ok := response.Response.(*log.AssertResponse)
	if assert.True(t, ok) {
		assert.NotNil(t, logValidatorAssertResponse)
		assert.Equal(t, 4, len(logValidatorAssertResponse.Validations))
		for i := 0; i < 4; i++ {
			assert.Equal(t, 0, len(logValidatorAssertResponse.Validations[i].Failures))
			if !assert.Nil(t, logValidatorAssertResponse.Validations[i].Failures) {
				assert.FailNow(t, toolbox.AsString(i)+" "+logValidatorAssertResponse.Validations[i].Failures[0].Message)
			}
		}
		response = service.Run(context, &log.AssertRequest{
			Expect: []*log.TypedRecord{
				{
					Type: "t",
					Records: []interface{}{
						map[string]interface{}{
							"k5": "— -20",
						},
					},
				},
			},
		})

		assert.Equal(t, "", response.Error)
		logValidatorAssertResponse, ok = response.Response.(*log.AssertResponse)
		assert.True(t, ok)
		assert.NotNil(t, logValidatorAssertResponse)
		if !assert.Equal(t, 0, len(logValidatorAssertResponse.Validations[0].Failures)) {
			if !assert.Nil(t, logValidatorAssertResponse.Validations[0].Failures) {
				assert.FailNow(t, toolbox.AsString(0)+" "+logValidatorAssertResponse.Validations[0].Failures[0].Message)
			}
		}

	}
	{
		response = service.Run(context, &log.ResetRequest{
			LogTypes: []string{"t"},
		})
		assert.Equal(t, "", response.Error)
	}

}

var indexedLogRecords = `{"Timestamp":"2018-01-12T14:07:09.120207-08:00","EventType":"event1","EventID":"eeed0b0c-f7e4-11e7-b54f-784f438e6f38","ClientIP":"127.0.0.1:52141","ServerIP":"127.0.0.1:8777","Request":{"Method":"GET","URL":"http://127.0.0.1:8777/event1/?k1=v1\u0026k2=v2","Header":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]}},"Errors":""}
{"Timestamp":"2018-01-12T14:07:09.122259-08:00","EventType":"event1","EventID":"eeed4c70-f7e4-11e7-b54f-784f438e6f38","ClientIP":"127.0.0.1:52141","ServerIP":"127.0.0.1:8777","Request":{"Method":"GET","URL":"http://127.0.0.1:8777/event1/?k10=v1\u0026k2=v2","Header":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]}},"Errors":""}
{"Timestamp":"2018-01-12T14:07:09.123185-08:00","EventType":"event2","EventID":"eeed709c-f7e4-11e7-b54f-784f438e6f38","ClientIP":"127.0.0.1:52141","ServerIP":"127.0.0.1:8777","Request":{"Method":"GET","URL":"http://127.0.0.1:8777/event2/?k1=v1\u0026k2=v2","Header":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]}},"Errors":""}
{"Timestamp":"2018-01-12T14:07:09.123199-08:00","EventType":"event2","EventID":"eeed709c-f7e4-11e7-b54f-784f438e6f30","ClientIP":"127.0.0.1:52141","ServerIP":"127.0.0.1:8777","Request":{"Method":"GET","URL":"http://127.0.0.1:8777/event2/?k1=v1\u0026k2=v2","Header":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]}},"Errors":""}
`

func TestLogValidatorService_TestIndexedRecord(t *testing.T) {
	manager := endly.New()
	service, err := manager.Service(log.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()
	tempLog := path.Join(os.TempDir(), "endly_test_indexed.log")
	_ = toolbox.RemoveFileIfExist(tempLog)
	err = ioutil.WriteFile(tempLog, []byte(indexedLogRecords), 0644)
	assert.Nil(t, err)

	var response = service.Run(context, &log.ListenRequest{
		Source: location.NewResource(tempLog),
		Types: []*log.Type{
			{
				Name:         "t",
				Format:       "json",
				Mask:         "endly_test_indexed.log",
				IndexRegExpr: "\"EventID\":\"([^\"]+)\"",
			},
		},
	})
	assert.EqualValues(t, "", response.Error)

	response = service.Run(context, &log.AssertRequest{
		LogWaitTimeMs:     3000,
		LogWaitRetryCount: 3,
		Expect: []*log.TypedRecord{

			{
				Type: "t",
				Records: []interface{}{
					map[string]interface{}{
						"EventType": "event1",
						"EventID":   "eeed4c70-f7e4-11e7-b54f-784f438e6f38",
						"Timestamp": "2018-01-12T14:07:09.122259-08:00",
					},
					map[string]interface{}{
						"EventType": "event1",
						"EventID":   "eeed0b0c-f7e4-11e7-b54f-784f438e6f38",
						"Timestamp": "2018-01-12T14:07:09.120207-08:00",
					},
					map[string]interface{}{
						"Timestamp": "2018-01-12T14:07:09.123185-08:00",
						"EventType": "event2",
					},
					map[string]interface{}{
						"Timestamp": "2018-01-12T14:07:09.123185-08:00",
						"EventType": "event2",
						"EventID":   "eeed709c-f7e4-11e7-b54f-784f438e6f30",
					},
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)
	logValidatorAssertResponse, ok := response.Response.(*log.AssertResponse)
	if assert.True(t, ok) {
		if assert.NotNil(t, logValidatorAssertResponse) {
			assert.EqualValues(t, 4, len(logValidatorAssertResponse.Validations))
			for i := 0; i < 3; i++ {
				if !assert.EqualValues(t, 0, logValidatorAssertResponse.Validations[i].FailedCount) {
					assert.Fail(t, logValidatorAssertResponse.Validations[i].Failures[0].Message)
				}
			}
			assert.EqualValues(t, 1, logValidatorAssertResponse.Validations[3].FailedCount)

		}
	}

}

func Test_TT(t *testing.T) {
	fmt.Printf(regexp.QuoteMeta("events[v3].csv]"))
}

var templateLogUTF8 = map[string]interface{}{
	"dash":    "—",
	"english": "The quick brown fox jumps over the lazy dog.",
	"russian": "Быстрая коричневая лиса перепрыгнула через ленивую собаку.",
	"chinese": "你好",
	"polish":  "zażółć gęślą jaźń ZAŻÓŁĆ GĘŚLĄ JAŹŃ",
}

func TestLogWithUTF8ValidatorService_NewRequest(t *testing.T) {
	manager := endly.New()
	service, err := manager.Service(log.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()
	tempPath := path.Join(os.TempDir(), toolbox.AsString(time.Now().Unix()))
	err = os.Mkdir(tempPath, 0755)
	assert.Nil(t, err)
	var template, _ = toolbox.AsJSONText(templateLogUTF8)

	var fileURL = strings.Replace(location.NewResource(tempPath).URL, "file://", "scp://127.0.0.1", 1)
	credentials, err := util.GetDummyCredential()
	assert.Nil(t, err)

	var response = service.Run(context, &log.ListenRequest{
		Source: location.NewResource(fileURL, credentials),
		Types: []*log.Type{
			{
				Name:   "t",
				Format: "json",
				Mask:   "*.log",
			},
		},
	})

	var i = 0
	var logName = fmt.Sprintf("test%v.log", i)
	var fullLogname = path.Join(tempPath, logName)

	_ = toolbox.RemoveFileIfExist(fullLogname)
	var logContent = fmt.Sprintf(template)

	err = ioutil.WriteFile(fullLogname, []byte(logContent), 0644)
	if err != nil {
		assert.FailNow(t, fmt.Sprintf("%v", err))
	}
	time.Sleep(time.Second)

	//fmt.Println("File log path: ", fullLogname)

	assert.Equal(t, "", response.Error)
	var listenResponse, ok = response.Response.(*log.ListenResponse)
	if !assert.True(t, ok) {
		return
	}
	if !assert.NotNil(t, listenResponse) {
		return
	}

	logTypeMeta, ok := listenResponse.Meta["t"]
	assert.True(t, ok)
	assert.NotNil(t, logTypeMeta)
	assert.True(t, strings.HasSuffix(logTypeMeta.Source.URL, tempPath))
	assert.True(t, len(logTypeMeta.LogFiles) >= 1)

	response = service.Run(context, &log.AssertRequest{
		LogWaitTimeMs:     3000,
		LogWaitRetryCount: 3,
		Expect: []*log.TypedRecord{
			{
				Type: "t",
				Records: []interface{}{
					map[string]interface{}{
						"dash":    "—",
						"english": "The quick brown fox jumps over the lazy dog.",
						"russian": "Быстрая коричневая лиса перепрыгнула через ленивую собаку.",
						"chinese": "你好",
						"polish":  "zażółć gęślą jaźń ZAŻÓŁĆ GĘŚLĄ JAŹŃ",
					},
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)

	logValidatorAssertResponse, ok := response.Response.(*log.AssertResponse)

	if assert.True(t, ok) {
		assert.NotNil(t, logValidatorAssertResponse)
		assert.Equal(t, 1, len(logValidatorAssertResponse.Validations))
		for i := 0; i < 1; i++ {
			assert.Equal(t, 0, len(logValidatorAssertResponse.Validations[i].Failures))
			if !assert.Nil(t, logValidatorAssertResponse.Validations[i].Failures) {
				assert.FailNow(t, toolbox.AsString(i)+" "+logValidatorAssertResponse.Validations[i].Failures[0].Message)
			}

		}
	}
}
