package pubsub

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly/cloud/ec2"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"strings"
	"time"
)

//https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/sqs/sqs_createqueues.go

type awsPubSub struct {
	session *session.Session
	sqs     *sqs.SQS
	sns     *sns.SNS
	timeout time.Duration
}

func (c *awsPubSub) sendMessage(dest *Resource, message *Message) (Result, error) {
	queueURL, err := c.getQueueURL(dest.Name)
	if err != nil {
		return nil, err
	}
	input := &sqs.SendMessageInput{
		DelaySeconds:      aws.Int64(1),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{},
		QueueUrl:          &queueURL,
	}

	if len(message.Attributes) > 0 {
		for k, v := range message.Attributes {
			input.MessageAttributes[k] = &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
	}
	var body = toolbox.AsString(message.Data)
	input.MessageBody = aws.String(body)
	result, err := c.sqs.SendMessage(input)
	if err != nil {
		return nil, err
	}
	return *result.MessageId, nil
}

func (c *awsPubSub) publishMessage(dest *Resource, message *Message) (Result, error) {
	topicARN, err := c.getTopicARN(dest.Name)
	if err != nil {
		return nil, err
	}
	input := &sns.PublishInput{
		MessageAttributes: map[string]*sns.MessageAttributeValue{},
		TopicArn:          aws.String(topicARN),
	}
	if len(message.Attributes) > 0 {
		for k, v := range message.Attributes {
			input.MessageAttributes[k] = &sns.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
	}
	var body = toolbox.AsString(message.Data)
	input.Message = aws.String(body)
	input.Subject = aws.String(message.Subject)
	output, err := c.sns.Publish(input)
	if err != nil {
		return nil, err
	}
	return *output.MessageId, nil
}

func (c *awsPubSub) Push(dest *Resource, message *Message) (Result, error) {
	switch dest.Type {
	case ResourceTypeTopic:
		return c.publishMessage(dest, message)
	case ResourceTypeQueue:
		return c.sendMessage(dest, message)

	}
	return nil, fmt.Errorf("unsupported resource type: %v", dest.Type)
}

func (c *awsPubSub) PullN(source *Resource, count int) ([]*Message, error) {
	queueURL, err := c.getQueueURL(source.Name)
	if err != nil {
		return nil, err
	}
	input := &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(queueURL),
		AttributeNames: aws.StringSlice([]string{
			"All",
		}),
		MaxNumberOfMessages: aws.Int64(int64(count)),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: aws.Int64(int64(c.timeout * time.Second)),
	}
	// Receive a message from the SQS queue with long polling enabled.
	output, err := c.sqs.ReceiveMessage(input)
	var result = make([]*Message, 0)
	if err != nil || len(output.Messages) == 0 {
		return result, err
	}
	for _, msg := range output.Messages {
		message := &Message{
			ID:         *msg.MessageId,
			Attributes: map[string]string{},
		}
		if msg.Body != nil {
			message.Data = *msg.Body
		}
		if len(msg.MessageAttributes) > 0 {
			for k, v := range msg.MessageAttributes {
				val := ""
				if v != nil {
					val = *v.StringValue
				}
				message.Attributes[k] = val
			}
		}
	}
	return result, nil
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
	return &Resource{URL: *output.SubscriptionArn}, nil
}

func (c *awsPubSub) createQueue(resource *ResourceSetup) (*Resource, error) {
	var name = resource.Name

	if resource.Recreate {
		if _, err := c.getQueueURL(resource.Name); err == nil {
			if err = c.deleteQueue(&resource.Resource); err != nil {
				return nil, fmt.Errorf("failed to delete queue: %v, %v", name, err)
			}
		}
	}

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
	var resultResource = &Resource{URL: *result.QueueUrl, Name: name}
	if resource.Config.Topic != nil {
		topicURL, err := c.getTopicARN(resource.Config.Topic.URL)
		if err != nil {
			return nil, err
		}
		_, err = c.createSubscription(topicURL, *result.QueueUrl)
		if err != nil {
			return nil, err
		}
	}
	return resultResource, nil
}

func (c *awsPubSub) getTopicARN(topicURL string) (string, error) {
	input := &sns.ListTopicsInput{}
	for { //TODO look into better way to get topic URL
		output, err := c.sns.ListTopics(input)
		if err != nil {
			return "", err
		}
		for _, topic := range output.Topics {
			parts := strings.Split(*topic.TopicArn, ":")
			candidate := parts[len(parts)-1]
			if candidate == topicURL {
				return *topic.TopicArn, nil
			}
		}
		input.NextToken = output.NextToken
		if output.NextToken == nil {
			break
		}
	}
	return "", fmt.Errorf("failed to lookup topic URL %v", topicURL)
}

func (c *awsPubSub) getQueueURL(queueName string) (string, error) {
	result, err := c.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to lookup queue URL %v", queueName)
	}
	return *result.QueueUrl, nil
}

func (c *awsPubSub) createTopic(resource *ResourceSetup) (*Resource, error) {
	var name = resource.Name

	if resource.Recreate {
		if arn, _ := c.getTopicARN(resource.Name); arn != "" {
			if err := c.deleteTopic(&resource.Resource); err != nil {
				return nil, fmt.Errorf("failed to delete topic: %v, %v", name, err)
			}
		}
	}

	input := &sns.CreateTopicInput{
		Name: aws.String(name),
	}
	result, err := c.sns.CreateTopic(input)
	if err != nil {
		return nil, err
	}
	var resultResource = &Resource{URL: *result.TopicArn, Name: resource.Name}
	return resultResource, nil
}

func (c *awsPubSub) Create(resource *ResourceSetup) (*Resource, error) {
	switch resource.Type {
	case ResourceTypeTopic:
		return c.createTopic(resource)
	case ResourceTypeQueue:
		return c.createQueue(resource)
	}
	return nil, fmt.Errorf("unsupported resource type: %v", resource.Type)
}

func (c *awsPubSub) deleteQueue(resource *Resource) error {
	queueURL, err := c.getQueueURL(resource.Name)
	if err != nil {
		return err
	}
	_, err = c.sqs.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return nil
}

func (c *awsPubSub) deleteTopic(resource *Resource) error {
	queueURL, err := c.getTopicARN(resource.Name)
	if err != nil {
		return err
	}
	_, err = c.sns.DeleteTopic(&sns.DeleteTopicInput{
		TopicArn: aws.String(queueURL),
	})
	return nil
}

func (c *awsPubSub) Delete(resource *Resource) error {
	switch resource.Type {
	case ResourceTypeQueue:
		return c.deleteQueue(resource)
	case ResourceTypeTopic:
		return c.deleteTopic(resource)
	}
	return fmt.Errorf("unsupported resource type: %v", resource.Type)
}

func (c *awsPubSub) Close() error {
	return nil
}

func newAwsSqsClient(credConfig *cred.Config, timeout time.Duration) (Client, error) {
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
