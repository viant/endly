package reset_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	endpoint "github.com/viant/endly/endpoint/http"
	runner "github.com/viant/endly/runner/rest"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func StartRestTestServer(port int) error {
	baseDir := toolbox.CallerDirectory(3)
	return endpoint.StartServer(port, &endpoint.HTTPServerTrips{
		IndexKeys:     []string{endpoint.MethodKey, endpoint.URLKey, endpoint.BodyKey, endpoint.CookieKey, endpoint.ContentTypeKey},
		BaseDirectory: path.Join(baseDir, "test/send"),
	})
}

func TestResetRunnerService_Run(t *testing.T) {
	err := StartRestTestServer(8114)
	if !assert.Nil(t, err) {
		return
	}
	manager := endly.NewManager()
	service, err := manager.Service(runner.ServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	serviceResponse := service.Run(context, &runner.SendRequest{
		URL:     "http://127.0.0.1:8114/reverse",
		Method:  "POST",
		Request: "{\"text\":\"abc\"}",
	})

	assert.Equal(t, "", serviceResponse.Error)
	response, ok := serviceResponse.Response.(*runner.SendResponse)
	if assert.True(t, ok) {
		assert.NotNil(t, response)
		if toolbox.IsMap(response) {
			responseMap := toolbox.AsMap(response)
			assert.EqualValues(t, "cba", responseMap["text"])
		}
	}

}
