package pubsub

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly/cloud/ec2"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"time"
)

//https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/sqs/sqs_createqueues.go

type awsPubSub struct {
	session *session.Session
	sqs     *sqs.SQS
	sns     *sns.SNS
	timeout time.Duration
}

func (c *awsPubSub) Push(dest string, message *Message) (Result, error) {
	return nil, nil
}

func (c *awsPubSub) PullN(source string, count int) ([]*Message, error) {
	return nil, nil
}

func (c *awsPubSub) createSubscription(topicURL, queueURL string) (*Resource, error) {
	input := &sns.SubscribeInput{
		Endpoint:              aws.String(queueURL),
		Protocol:              aws.String("sqs"),
		TopicArn:              aws.String(topicURL),
		ReturnSubscriptionArn: aws.Bool(true),
	}
	output, err := c.sns.Subscribe(input)
	if err != nil {
		return nil, err
	}
	return &Resource{Resource: url.NewResource(*output.SubscriptionArn)}, nil
}

func (c *awsPubSub) createQueue(resource *Resource) (*Resource, error) {
	var name = ""
	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(name),
		Attributes: map[string]*string{},
	}

	if resource.Config != nil && len(resource.Config.Attributes) > 0 {
		for k, v := range resource.Config.Attributes {
			input.Attributes[k] = aws.String(v)
		}
	}
	result, err := c.sqs.CreateQueue(input)
	if err != nil {
		return nil, err
	}
	var resultResource = &Resource{Resource: url.NewResource(*result.QueueUrl)}

	if resource.Config.Topic != nil {

		//createSubscription
		//c.createTopic(resource.Config.Topic)
	}
	return resultResource, nil
}

func (c *awsPubSub) createTopic(resource *Resource) (*Resource, error) {
	var name = ""
	input := &sns.CreateTopicInput{
		Name: aws.String(name),
	}
	result, err := c.sns.CreateTopic(input)
	if err != nil {
		return nil, err
	}
	var resultResource = &Resource{Resource: url.NewResource(*result.TopicArn)}
	return resultResource, nil
}

func (c *awsPubSub) Create(resource *Resource) (*Resource, error) {

	return nil, nil
}

func (c *awsPubSub) Delete(resource *Resource) error {
	return nil
}

func (c *awsPubSub) Close() error {
	return nil
}

func newAwsSqsClient(credConfig *cred.Config, URL string, timeout time.Duration) (Client, error) {
	config, err := ec2.GetAWSCredentialConfig(credConfig)
	if err != nil {
		return nil, err
	}
	var client = &awsPubSub{
		timeout: timeout,
	}
	if client.session, err = session.NewSession(config); err != nil {
		return nil, err
	}
	client.sqs = sqs.New(client.session)
	client.sns = sns.New(client.session)
	return client, nil
}
