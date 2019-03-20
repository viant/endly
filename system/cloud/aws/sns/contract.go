package sns

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sns"
)

type SetupTopicInput sns.CreateTopicInput

type SetupSubscriptionInput struct {
	*sns.SubscribeInput `json:",inline"`
	Topic               *string
}

func (i *SetupSubscriptionInput) Validate() error {
	if i.Topic == nil && (i.SubscribeInput == nil || i.TopicArn == nil) {
		return fmt.Errorf("topic was empty")
	}
	if i.SubscribeInput == nil {
		return fmt.Errorf("protocol was empty")
	}
	if i.Endpoint == nil {
		return fmt.Errorf("endpoint was empty")
	}
	return nil
}
