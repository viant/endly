package cloudfunctions

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"google.golang.org/api/cloudfunctions/v1"
	"os"
	"path"
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	context := endly.New().NewContext(nil)
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/am.json")) {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "am",
	})
	if !assert.Nil(t, err) {
		return
	}

	credConfig, err := gc.InitCredentials(context, map[string]interface{}{
		"Credentials": "am",
	})
	if !assert.Nil(t, err) {
		return
	}

	request, err := context.NewRequest(ServiceID, "operationsList", map[string]interface{}{
		"urlParams": map[string]interface{}{
			"filter": fmt.Sprintf("project:%s,latest:true", credConfig.ProjectID),
		},
	})

	response := make(map[string]interface{})
	err = endly.Run(context, request, &response)
	assert.Nil(t, err)
	toolbox.DumpIndent(response, true)
}



func TestService_Deploy(t *testing.T) {
	context := endly.New().NewContext(nil)
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/am.json")) {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "am",
	})
	if !assert.Nil(t, err) {
		return
	}
	srv := &service{}

	parent := toolbox.CallerDirectory(3)

	response, err := srv.Deploy(context, &DeployRequest{
		CloudFunction: &cloudfunctions.CloudFunction{
			Name:         "HelloWorld",
			EntryPoint:   "HelloWorld",
			Runtime:      "go111",
			HttpsTrigger: &cloudfunctions.HttpsTrigger{},
		},
		Location: "us-central1",
		Source:   url.NewResource(path.Join(parent, "test/hello.zip")),
	})
	fmt.Printf("%v\n", time.Now())
	if !assert.Nil(t, err) {
		return
	}

	toolbox.DumpIndent(response, true)

}
