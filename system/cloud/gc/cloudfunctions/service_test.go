package cloudfunctions

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func TestNew(t *testing.T) {

	context := endly.New().NewContext(nil)
	if ! toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/am.json")) {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials":"am",
	})
	if ! assert.Nil(t, err) {
		return
	}


	credConfig,err := gc.InitCredentials(context,  map[string]interface{}{
		"Credentials":"am",
	})
	if ! assert.Nil(t, err) {
		return
	}

	request, err := context.NewRequest(ServiceID, "operationsList", map[string]interface{}{
		"urlParams":map[string]interface{}{
			"filter": 	fmt.Sprintf("project:%s,latest:true",credConfig.ProjectID),
		},
	})

	response := make(map[string]interface{})
	err = endly.Run(context, request, &response)
	assert.Nil(t, err)
}