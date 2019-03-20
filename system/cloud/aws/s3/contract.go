package s3

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/viant/endly/system/cloud/aws/lambda"
)

//SetupBucketNotificationInput setup permission for specified bucket
type SetupBucketNotificationInput struct {
	Bucket                     *string
	*NotificationConfiguration `json:",inline"`
}

type LambdaFunctionConfiguration struct {
	s3.LambdaFunctionConfiguration `yaml:",inline" json:",inline"`
	Filter                         *NotificationFilter
	lambda.SetupPermissionInput
}

type NotificationConfiguration struct {
	*s3.NotificationConfiguration
	LambdaFunctionConfigurations []*LambdaFunctionConfiguration
}

type SetupBucketNotificationOutput struct {
	Bucket                        *string
	*s3.NotificationConfiguration `json:",inline"`
	Permissions                   []*lambda.SetupPermissionInput
}

func (i *SetupBucketNotificationInput) Init() error {
	if i.Bucket == nil {
		return nil
	}
	if i.NotificationConfiguration == nil {
		bucketNotification := &s3.NotificationConfiguration{}
		i.NotificationConfiguration = &NotificationConfiguration{
			NotificationConfiguration: bucketNotification,
		}
	}
	if len(i.NotificationConfiguration.LambdaFunctionConfigurations) == 0 {
		i.NotificationConfiguration.LambdaFunctionConfigurations = make([]*LambdaFunctionConfiguration, 0)
	}
	for _, lambdaConfig := range i.NotificationConfiguration.LambdaFunctionConfigurations {
		if lambdaConfig.SourceArn == nil {
			lambdaConfig.SourceArn = aws.String("arn:aws:s3:::" + *i.Bucket)
		}
		if lambdaConfig.Id == nil {
			lambdaConfig.Id = aws.String("${uuid.next}")
		}
		if err := lambdaConfig.Init(); err != nil {
			return err
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
