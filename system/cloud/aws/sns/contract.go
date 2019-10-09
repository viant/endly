package sns

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	currentAccount = "${aws.accountID}"
)

//SetupTopicInput represents setup topic request
type SetupTopicInput sns.CreateTopicInput

//SetupSubscriptionInput represets setup subscription request
type SetupSubscriptionInput struct {
	*sns.SubscribeInput `json:",inline"`
	Topic               *string
}

//SetupPermissionInput creates a permission if it does not exists
type SetupPermissionInput struct {
	sns.AddPermissionInput
	Everybody bool
	SourceArn string
	Topic     string
}

//SetupSubscriptionInput check if request is valid
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

func (i *SetupPermissionInput) Init() error {
	if i.Label == nil {
		source := ""
		if ARN, err := arn.Parse(i.SourceArn); err == nil {
			source = ARN.Resource + "-"
		}
		i.Label = aws.String(fmt.Sprintf("%s%vPermission", source, i.Topic))
	}
	if len(i.AWSAccountId) == 0 {
		i.AWSAccountId = []*string{aws.String(currentAccount)}
	}
	return nil

}

func (i *SetupPermissionInput) Validate() error {
	if i.Topic == "" && i.TopicArn == nil {
		return errors.New("topic was empty")
	}
	if len(i.ActionName) == 0 {
		return errors.New("actionNames was empty")
	}
	return nil
}
