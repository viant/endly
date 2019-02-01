package msg

import (
	"fmt"
	"github.com/viant/endly"
	"time"
)

const (
	ResourceVendorGoogleCloud      = "gc"
	ResourceVendorGoogleCloudPlatform      = "gcp"
	ResourceVendorAmazonWebService = "aws"
)

type Client interface {
	Push(dest *Resource, message *Message) (Result, error)

	PullN(source *Resource, count int, nack bool) ([]*Message, error)

	SetupResource(resource *ResourceSetup) (*Resource, error)

	DeleteResource(resource *Resource) error

	Close() error
}

//NewPubSubClient creates a new Client
func NewPubSubClient(context *endly.Context, dest *Resource, timeout time.Duration) (Client, error) {
	credConfig, err := context.Secrets.GetCredentials(dest.Credentials)
	if err != nil {
		return nil, err
	}
	if dest.Vendor == "" {
		dest.Vendor = inferResourceTypeFromCredentialConfig(credConfig)
	}
	state := context.State()
	if credConfig.ProjectID != "" {
		state.SetValue("msg.projectID", credConfig.ProjectID)
	}
	dest = expandResource(context, dest)
	switch dest.Vendor {
	case ResourceVendorGoogleCloud, ResourceVendorGoogleCloudPlatform:
		return newCloudPubSub(credConfig, dest.URL, timeout)
	case ResourceVendorAmazonWebService:
		return newAwsSqsClient(credConfig, timeout)
	}
	return nil, fmt.Errorf("unsupported vendor: '%v'", dest.Vendor)

}
