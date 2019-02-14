package bigquery

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
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
}

func Test_Meta(t *testing.T) {

	context := endly.New().NewContext(nil)
	assert.NotNil(t, context)


	parent := toolbox.CallerDirectory(3)

	resource := url.NewResource(path.Join(parent, "mv.yaml"))

	rawRequest := map[string]interface{}{}


	resource.Decode(&rawRequest)

	if normalized, err := toolbox.NormalizeKVPairs(rawRequest);err == nil{
		rawRequest  = toolbox.AsMap(normalized)
	}

	request, err := context.NewRequest(ServiceID, "tablesInsert",rawRequest)
	assert.Nil(t, err)

	fmt.Printf("%T\n", request)
	err =endly.Run(context, request, nil)

	assert.Nil(t, err)


}
