package msg

import (
	"context"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/cred"
	"time"
)

const (
	ResourceVendorGoogleCloudPlatform = "gcp"
	ResourceVendorAmazonWebService    = "aws"
	ResourceVendorKafka               = "kafka"
)

type Client interface {
	Push(ctx context.Context, dest *Resource, message *Message) (Result, error)

	PullN(ctx context.Context, source *Resource, count int, nack bool) ([]*Message, error)

	SetupResource(resource *ResourceSetup) (*Resource, error)

	DeleteResource(resource *Resource) error

	Close() error
}

// NewPubSubClient creates a new Client
func NewPubSubClient(context *endly.Context, dest *Resource, timeout time.Duration) (Client, error) {

	credConfig := &cred.Config{}
	var err error
	if dest.Credentials != "" {
		credConfig, err = context.Secrets.GetCredentials(dest.Credentials)
	}
	if err != nil {
		return nil, err
	}
	if len(dest.Brokers) > 0 {
		dest.Vendor = ResourceVendorKafka
		dest.Type = ResourceTypeTopic
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
	case ResourceVendorGoogleCloudPlatform:
		return newCloudPubSub(credConfig, dest.URL, timeout)
	case ResourceVendorAmazonWebService:
		return newAwsSqsClient(credConfig, timeout)
	case ResourceVendorKafka:
		return newKafkaClient(timeout)
	}
	return nil, fmt.Errorf("unsupported vendor: '%v'", dest.Vendor)

}
