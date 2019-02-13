package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"google.golang.org/api/pubsub/v1"
	"os"
	"path"
	"testing"
)

func TestClient(t *testing.T) {
	context := endly.New().NewContext(nil)
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "4234234dasdasde",
	})
	assert.NotNil(t, err)
	_, err = GetClient(context)
	assert.NotNil(t, err)
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/am.json")) {
		return
	}
	err = InitRequest(context, map[string]interface{}{
		"Credentials": "am",
	})
	assert.Nil(t, err)
	client, err := GetClient(context)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	service, ok := client.Service().(*pubsub.Service)
	if !assert.True(t, ok) {
		return
	}
	assert.NotNil(t, service)
	instance := service.Projects.Topics.List("projects/abstractmeta-p1")
	assert.NotNil(t, instance)
	output, err := instance.Do()
	fmt.Printf("%v %v\n", output, err)
	toolbox.DumpIndent(output, true)
}
