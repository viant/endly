package bigquery

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func TestNew(t *testing.T) {

	context := endly.New().NewContext(nil)
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/am.json")) {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "am",
	})
	assert.Nil(t, err)
	request, err := context.NewRequest(ServiceID, "instancesList", map[string]interface{}{
		"zone": "us-central1-f",
	})
	assert.Nil(t, err)
	assert.NotNil(t, request)
}

func Test_Meta(t *testing.T) {

	context := endly.New().NewContext(nil)
	//client := &bigquery.Service{}
	assert.NotNil(t, context)

	jjj := &QueryRequest{}
	toolbox.InitStruct(jjj)

}
