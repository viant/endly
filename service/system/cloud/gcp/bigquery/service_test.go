package bigquery

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func TestNew(t *testing.T) {

	context := endly.New().NewContext(nil)
	if !gcp.HasTestCredentials() {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)
}

func Test_Meta(t *testing.T) {

	context := endly.New().NewContext(nil)
	assert.NotNil(t, context)

	parent := toolbox.CallerDirectory(3)

	resource := location.NewResource(path.Join(parent, "mv.yaml"))

	rawRequest := map[string]interface{}{}

	resource.Decode(&rawRequest)

	if normalized, err := toolbox.NormalizeKVPairs(rawRequest); err == nil {
		rawRequest = toolbox.AsMap(normalized)
	}

	request, err := context.NewRequest(ServiceID, "tablesInsert", rawRequest)
	assert.Nil(t, err)

	fmt.Printf("%T\n", request)
	err = endly.Run(context, request, nil)

	assert.Nil(t, err)

}
