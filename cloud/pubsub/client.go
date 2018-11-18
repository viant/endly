package pubsub

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"

	"time"
)

const (
	//GoogleCloudPubSubSchema represents endly resource for google cloud pubsub resource i.e gcpubsub:/projects/[PROJECT ID]/subscriptions/[SUBSCRIPTION]
	GoogleCloudPubSubSchema = "gcpubsub"
)

type Client interface {
	Push(dest string, message *Message) (Result, error)

	PullN(source string, count int) ([]*Message, error)

	Create(resource *Resource) (*Resource, error)

	Delete(resource *Resource) error

	Close() error
}

//NewPubSubClient creates a new Client
func NewPubSubClient(context *endly.Context, dest *url.Resource, timeout time.Duration) (Client, error) {
	config, err := context.Secrets.GetCredentials(dest.Credentials)
	if err != nil {
		return nil, err
	}
	state := context.State()
	if config.ProjectID != "" {
		state.SetValue("pubsub.projectID", config.ProjectID)
	}

	var destURL = state.ExpandAsText(dest.URL)
	if dest.ParsedURL.Scheme == GoogleCloudPubSubSchema {
		return newCloudPubSub(config, destURL, timeout)
	}

	return nil, fmt.Errorf("unsupported scheme: '%v'", dest.ParsedURL.Scheme)

}
