package s3

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/viant/endly/service/system/cloud/aws/lambda"
	"github.com/viant/endly/service/system/cloud/aws/sns"
	"github.com/viant/endly/service/system/cloud/aws/sqs"
)

const (
	sendMessageAction = "sendMessage"
	publishAction     = "publish"
)

// SetupBucketNotificationInput setup permission for specified bucket
type SetupBucketNotificationInput struct {
	Bucket                     *string
	*NotificationConfiguration `json:",inline"`
}

type LambdaFunctionConfiguration struct {
	s3.LambdaFunctionConfiguration `yaml:",inline" json:",¬inline"`
	Filter                         *NotificationFilter `yaml:"-"`
	lambda.SetupPermissionInput
}

type QueueConfiguration struct {
	s3.QueueConfiguration
	Filter *NotificationFilter
	Queue  string
	sqs.SetupPermissionInput
}

type TopicConfiguration struct {
	s3.TopicConfiguration
	Filter *NotificationFilter
	sns.SetupPermissionInput
	Topic string
}

type NotificationConfiguration struct {
	QueueConfigurations          []*QueueConfiguration          `locationName:"QueueConfiguration" type:"list" flattened:"true"`
	TopicConfigurations          []*TopicConfiguration          `locationName:"TopicConfiguration" type:"list" flattened:"true"`
	LambdaFunctionConfigurations []*LambdaFunctionConfiguration `locationName:"LambdaFunctionConfigurations" type:"list" flattened:"true"`
}

type SetupBucketNotificationOutput struct {
	Bucket                        *string
	*s3.NotificationConfiguration `json:",inline"`
	LambdaPermissions             []*lambda.SetupPermissionInput
	QueuePermissions              []*sqs.SetupPermissionInput
}

func (i *SetupBucketNotificationInput) Init() error {
	if i.Bucket == nil {
		return nil
	}
	if i.NotificationConfiguration == nil {
		i.NotificationConfiguration = &NotificationConfiguration{}
	}
	if len(i.NotificationConfiguration.LambdaFunctionConfigurations) > 0 {
		for _, config := range i.NotificationConfiguration.LambdaFunctionConfigurations {
			if config.SourceArn == nil {
				config.SourceArn = aws.String("arn:aws:s3:::" + *i.Bucket)
			}
			if err := config.Init(); err != nil {
				return err
			}
		}
	}

	if len(i.NotificationConfiguration.QueueConfigurations) > 0 {
		for j := range i.NotificationConfiguration.QueueConfigurations {
			config := i.NotificationConfiguration.QueueConfigurations[j]
			if config.Queue == "" && config.QueueArn == nil {
				return errors.New("queue was empty")
			}
			if len(config.Actions) == 0 {
				config.Actions = []*string{
					aws.String(sendMessageAction),
				}
			}
		}
	}

	if len(i.NotificationConfiguration.TopicConfigurations) > 0 {
		for j := range i.NotificationConfiguration.TopicConfigurations {
			config := i.NotificationConfiguration.TopicConfigurations[j]
			if config.Topic == "" && config.TopicArn == nil {
				return errors.New("topic was empty")
			}
			if len(config.ActionName) == 0 {
				config.ActionName = []*string{
					aws.String(publishAction),
				}
			}
		}
	}

	return nil
}

func (i *LambdaFunctionConfiguration) Init() error {
	if i.Principal == nil {
		i.Principal = aws.String("s3.amazonaws.com")
	}
	if i.FunctionName != nil {
		if i.Action == nil {
			i.Action = aws.String("lambda:InvokeFunction")
		}
	}
	if i.StatementId == nil {
		i.StatementId = aws.String("${uuid.next}")
	}
	return nil
}

func (i *SetupBucketNotificationInput) Validate() error {
	if i.Bucket == nil {
		return errors.New("bucket was empty")
	}
	return nil
}
