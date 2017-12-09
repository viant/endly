package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
	"path"
)

func StartRestTestServer(port int) error {
	baseDir := toolbox.CallerDirectory(3)
	return endly.StartHttpServer(port, &endly.HttpServerTrips{
		IndexKeys:[]string{endly.MethodKey, endly.URLKey, endly.BodyKey, endly.CookieKey, endly.ContentTypeKey},
		BaseDirectory:path.Join(baseDir, "test/rest/send"),
	})
}


func TestResetRunnerService_Run(t *testing.T) {
	err := StartRestTestServer(8114)
	if ! assert.Nil(t, err) {
		return
	}
	manager := endly.NewManager()
	service, err := manager.Service(endly.RestServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())
	serviceResponse := service.Run(context, &endly.RestSendRequest{
				URL: "http://127.0.0.1:8114/reverse",
				Method:"POST",
				Request:"{\"text\":\"abc\"}",
	})

	assert.Equal(t, "", serviceResponse.Error)
	response, ok := serviceResponse.Response.(*endly.RestSendResponse)
	if assert.True(t, ok) {
		assert.NotNil(t, response)
		if toolbox.IsMap(response) {
			responseMap := toolbox.AsMap(response)
			assert.EqualValues(t, "cba", responseMap["text"])
		}
	}





}

