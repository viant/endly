package endly_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

var templateLog = map[string]interface{}{
	"k1": "v1",
	"k2": []string{"1", "2", "%v"},
	"k3": 123,
	"k4": map[string]interface{}{
		"s1": 1,
		"s2": "%v",
	},
	"k5": "%v",
}

func GetMapAsString(source interface{}) string {
	buf := new(bytes.Buffer)
	toolbox.NewJSONEncoderFactory().Create(buf).Encode(source)
	return buf.String()
}

func BuildLogContent(from, to, multiplier int, template string) string {
	var result = make([]string, 0)
	for i := from; i <= to; i++ {
		result = append(result, fmt.Sprintf(template, multiplier*i, (100*1)+1, 10*i))
	}
	return strings.Join(result, "")
}

func TestLogValidatorService_NewRequest(t *testing.T) {

	manager := endly.NewManager()
	service, err := manager.Service(endly.LogValidatorServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()
	tempPath := path.Join(os.TempDir(), toolbox.AsString(time.Now().Unix()))
	err = os.Mkdir(tempPath, 0755)
	assert.Nil(t, err)
	var template = GetMapAsString(templateLog)

	var response = service.Run(context, &endly.LogValidatorListenRequest{
		Source: url.NewResource(tempPath),
		Types: []*endly.LogType{
			{
				Name:   "t",
				Format: "json",
				Mask:   "*.log",
			},
		},
	})

	time.Sleep(time.Second)
	for i := 0; i < 2; i++ {
		var logName = fmt.Sprintf("test%v.log", i)
		var fullLogname = path.Join(tempPath, logName)

		toolbox.RemoveFileIfExist(fullLogname)
		var logContent = BuildLogContent(1, 3, i+1, template)
		err = ioutil.WriteFile(fullLogname, []byte(logContent), 0644)
		if err != nil {
			assert.FailNow(t, fmt.Sprintf("%v", err))
		}
	}

	time.Sleep(time.Second)
	assert.Equal(t, "", response.Error)
	var listenResponse, ok = response.Response.(*endly.LogValidatorListenResponse)
	assert.True(t, ok)
	assert.NotNil(t, listenResponse)

	logTypeMeta, ok := listenResponse.Meta["t"]
	assert.True(t, ok)
	assert.NotNil(t, logTypeMeta)
	assert.True(t, strings.HasSuffix(logTypeMeta.Source.URL, tempPath))
	assert.Equal(t, 2, len(logTypeMeta.LogFiles))

	response = service.Run(context, &endly.LogValidatorAssertRequest{
		ExpectedLogRecords: []*endly.ExpectedLogRecord{

			{
				Type: "t",
				Records: []map[string]interface{}{
					{
						"k5": "10",
					},
					{
						"k5": "20",
					},
					{
						"k5": "30",
					},
					{
						"k5": "10",
					},
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)
	assertionInfo, ok := response.Response.(*endly.AssertionInfo)
	assert.True(t, ok)
	assert.NotNil(t, assertionInfo)
	assert.Equal(t, 0, len(assertionInfo.TestFailed))

	response = service.Run(context, &endly.LogValidatorAssertRequest{
		ExpectedLogRecords: []*endly.ExpectedLogRecord{

			{

				Type: "t",
				Records: []map[string]interface{}{
					{
						"k5": "20",
					},
					{
						"k5": "30",
					},
				},
			},
		},
	})

	assert.Equal(t, "", response.Error)
	assertionInfo, ok = response.Response.(*endly.AssertionInfo)
	assert.True(t, ok)
	assert.NotNil(t, assertionInfo)
	assert.Equal(t, 0, len(assertionInfo.TestFailed))

}
