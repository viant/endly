package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"net/http"
	"testing"
	"time"
)

func StartTestServer(port string) {
	http.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
		var cookies endly.Cookies = reader.Cookies()
		var cookieValue = ""
		indexedCookies := cookies.IndexByName()
		if cookie, has := indexedCookies["ug"]; has {
			cookieValue = cookie.Value
		}
		if cookieValue != "" {
			writer.Header().Set("Set-Cookie", "ug=123")
		}
		writer.Write([]byte(fmt.Sprintf("Cookie:%v; Path:%v;", cookieValue, reader.URL.Path)))

	})
	http.ListenAndServe(":"+port, nil)
}

func TestHttpRunnerService_Run(t *testing.T) {
	go StartTestServer("8766")
	manager := endly.NewServiceManager()
	service, err := manager.Service(endly.HttpRunnerServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	time.Sleep(500 * time.Millisecond)
	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.SendRequest{
		Requests: []*endly.HttpRequest{
			{
				URL: "http://127.0.0.1:8766/abc",
				Cookies: []*http.Cookie{
					{
						Name:  "ug",
						Value: "191817161514",
					},
				},
				Extraction: []*endly.DataExtraction{
					{
						Name:    "path1",
						RegExpr: "Path:([^;]+)",
					},
					{
						Name:    "cookie1",
						RegExpr: "Cookie:([^;]+)",
					},
				},
			},
			{
				URL: "http://127.0.0.1:8766/xyz",
				Extraction: []*endly.DataExtraction{
					{
						Name:    "path2",
						RegExpr: "Path:([^;]+)",
					},
					{
						Name:    "cookie2",
						RegExpr: "Cookie:([^;]+)",
					},
				},
			},
			{
				MatchBody: "123",
				URL:       "http://127.0.0.1:8766/xxx",
				Cookies: []*http.Cookie{
					{
						Name:  "ug",
						Value: "33333",
					},
				},
				Extraction: []*endly.DataExtraction{
					{
						Name:    "path3",
						RegExpr: "Path:([^;]+)",
					},
					{
						Name:    "cookie3",
						RegExpr: "Cookie:([^;]+)",
					},
				},
			},
		},
	})
	assert.Nil(t, response.Error)
	sendResponse, ok := response.Response.(*endly.SendResponse)
	assert.True(t, ok)
	path1, _ := sendResponse.Extracted["path1"]
	assert.Equal(t, "/abc", path1)

	cookie1, _ := sendResponse.Extracted["cookie1"]
	assert.Equal(t, "191817161514", cookie1)

	path2, _ := sendResponse.Extracted["path2"]
	assert.Equal(t, "/xyz", path2)

	cookie2, _ := sendResponse.Extracted["cookie2"]
	assert.Equal(t, "123", cookie2)

	path3, _ := sendResponse.Extracted["path3"]
	assert.Equal(t, "/xxx", path3)

	cookie3, _ := sendResponse.Extracted["cookie3"]
	assert.Equal(t, "33333", cookie3)

}
