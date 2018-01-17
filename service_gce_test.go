package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"google.golang.org/api/compute/v1"
	"os"
	"path"
	"testing"
)

//https://cloud.google.com/compute/docs/reference/beta/instances/start

func TestNewGceService(t *testing.T) {

	credential := path.Join(os.Getenv("HOME"), ".secret/gce.json")

	if toolbox.FileExists(credential) && os.Getenv("GCE_PROJECT") != "" {
		manager := endly.NewManager()
		context := manager.NewContext(toolbox.NewContext())
		service, _ := context.Service(endly.GCEServiceID)

		project := os.Getenv("GCE_PROJECT")
		zone := "us-west1-b"
		instance := "instance-1"
		serviceResponse := service.Run(context, &endly.GCECallRequest{
			Credential: credential,
			Service:    "Instances",
			Method:     "Get1",
			Parameters: []interface{}{project, zone, instance},
		})
		assert.Equal(t, "", serviceResponse.Error)
		if gceResponse, ok := serviceResponse.Response.(*endly.GCECallResponse); ok && gceResponse != nil {
			if instance, ok := gceResponse.Response.(*compute.Instance); ok {
				assert.EqualValues(t, "instance-1", instance.Name)
			}

		}

	}
}
