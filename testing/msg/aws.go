package msg

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	eaws "github.com/viant/endly/system/cloud/aws"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"strings"
	"time"
)

type awsClient struct {
	config  *aws.Config
	session *session.Session
	sqs     *sqs.SQS
	sns     *sns.SNS
	timeout time.Duration
}

func (c *awsClient) sendMessage(dest *Resource, message *Message) (Result, error) {
	queueURL, err := c.getQueueURL(dest.Name)
	if err != nil {
		return nil, err
	}
	input := &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(1),
		QueueUrl:     &queueURL,
	}
	if len(message.Attributes) > 0 {
		input.MessageAttributes = make(map[string]*sqs.MessageAttributeValue)
		putSqsMessageAttributes(message.Attributes, input.MessageAttributes)
	}
	var body = toolbox.AsString(message.Data)
	input.MessageBody = aws.String(body)
	result, err := c.sqs.SendMessage(input)
	if err != nil {
		return nil, err
	}
	return *result.MessageId, nil
}

func (c *awsClient) publishMessage(dest *Resource, message *Message) (Result, error) {
	topicARN, err := c.getTopicARN(dest.Name)
	if err != nil {
		return nil, err
	}
	input := &sns.PublishInput{
		MessageAttributes: map[string]*sns.MessageAttributeValue{},
		TopicArn:          aws.String(topicARN),
	}
	if len(message.Attributes) > 0 {
		input.MessageAttributes = make(map[string]*sns.MessageAttributeValue)
		putSnsMessageAttributes(message.Attributes, input.MessageAttributes)
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

func (c *awsClient) Push(ctx context.Context, dest *Resource, message *Message) (Result, error) {
	switch dest.Type {
	case ResourceTypeTopic:
		return c.publishMessage(dest, message)
	case ResourceTypeQueue:
		return c.sendMessage(dest, message)

	}
	return nil, fmt.Errorf("unsupported resource type: %v", dest.Type)
}

func (c *awsClient) PullN(ctx context.Context, source *Resource, count int, nack bool) ([]*Message, error) {
	queueURL, err := c.getQueueURL(source.Name)
	if err != nil {
		return nil, err
	}
	var result = make([]*Message, 0)
	waitTime := int64(c.timeout / time.Second)
	if waitTime > 20 {
		waitTime = 20
	}
	if err := c.processMessages(queueURL, !nack, true, count, waitTime, func(msg *sqs.Message) (bool, error) {
		message := buildMessage(msg)
		result = append(result, message)
		return len(result) < count, nil

	}); err != nil {
		return nil, err
	}
	return result, nil
}

func buildReceiveMessageInput(queueURL string, pullCount int, waitTime int64, includeAttr bool) *sqs.ReceiveMessageInput {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: aws.Int64(int64(pullCount)),
		WaitTimeSeconds:     aws.Int64(waitTime),
		VisibilityTimeout:   aws.Int64(1),
	}
	if includeAttr {
		input.MessageAttributeNames = aws.StringSlice([]string{"All"})
		input.AttributeNames = aws.StringSlice([]string{"All"})
	}
	return input
}

func buildMessage(msg *sqs.Message) *Message {
	message := &Message{
		ID:         *msg.MessageId,
		Attributes: make(map[string]interface{}),
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
	return message
}

func (c *awsClient) createSubscription(topicURL, queueURL string) (*Resource, error) {
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

func (c *awsClient) processMessages(queueURL string, delete, includeAttributes bool, maxCount int, waitTimeSec int64, handler func(message *sqs.Message) (bool, error)) error {
	count := maxCount
	if count == 0 {
		input := &sqs.GetQueueAttributesInput{
			QueueUrl:       &queueURL,
			AttributeNames: []*string{aws.String("All")},
		}
		result, err := c.sqs.GetQueueAttributes(input)
		if err != nil {
			return errors.Wrapf(err, "failed to clean queue messages: %v", queueURL)
		}
		value := result.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessages]
		if value == nil {
			return nil
		}
		count := toolbox.AsInt(*value)

		if count == 0 {
			return nil
		}
	}

	isTerminated := false
	pullCount := 10
	for i := 0; i < count && !isTerminated; {
		if pullCount+i > count {
			pullCount = count % 10
		}
		receivedInput := buildReceiveMessageInput(queueURL, pullCount, waitTimeSec, includeAttributes)
		output, err := c.sqs.ReceiveMessage(receivedInput)
		if err != nil {
			return errors.Wrapf(err, "failed to clean queue messages: %v", queueURL)
		}
		i += len(output.Messages)
		time.Sleep(time.Second + 1)
		deleteInput := &sqs.DeleteMessageBatchInput{
			Entries:  make([]*sqs.DeleteMessageBatchRequestEntry, 0),
			QueueUrl: aws.String(queueURL),
		}

		for _, msg := range output.Messages {
			if delete {
				deleteInput.Entries = append(deleteInput.Entries, &sqs.DeleteMessageBatchRequestEntry{
					Id:            msg.MessageId,
					ReceiptHandle: msg.ReceiptHandle,
				})
			}
			if handler != nil {
				toContinue, err := handler(msg)
				if err != nil {
					return err
				}
				if !toContinue {
					isTerminated = true
					break
				}
			}
		}
		if len(deleteInput.Entries) > 0 {
			_, err = c.sqs.DeleteMessageBatch(deleteInput)
		}
		if err != nil {
			return errors.Wrapf(err, "failed to clean queue messages: %v", queueURL)
		}
	}
	return nil
}


func (c *awsClient) createQueue(resource *ResourceSetup) (*Resource, error) {
	var name = resource.Name
	queueURL, _ := c.getQueueURL(resource.Name)
	if resource.Recreate {
		if queueURL != "" {
			if err := c.deleteQueue(&resource.Resource); err != nil {
				return nil, fmt.Errorf("failed to delete queue: %v, %v", name, err)
			}
			time.Sleep(time.Minute + 1)
		}
	} else if queueURL != "" {
		//process and deletes outstanding messages
		if err := c.processMessages(queueURL, true, false, 0, 20, nil); err != nil {
			return nil, err
		}
	}

	input := &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	}
	if resource.Config != nil && len(resource.Config.Attributes) > 0 {
		input.Attributes = make(map[string]*string)
		for k, v := range resource.Config.Attributes {
			input.Attributes[k] = aws.String(v)
		}
	}
	result, err := c.sqs.CreateQueue(input)
	if err != nil {
		return nil, err
	}
	var resultResource = &Resource{URL: *result.QueueUrl, Name: name}
	if resource.Config != nil && resource.Config.Topic != nil {
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

func (c *awsClient) getTopicARN(topicURL string) (string, error) {
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


func (c *awsClient) getQueueURL(queueName string) (string, error) {
	result, err := c.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to lookup queue URL %v", queueName)
	}
	return *result.QueueUrl, nil
}

func (c *awsClient) createTopic(resource *ResourceSetup) (*Resource, error) {
	var name = resource.Name
	if resource.Recreate {
		if arn, _ := c.getTopicARN(resource.Name); arn != "" {
			if err := c.deleteTopic(&resource.Resource); err != nil {
				return nil, fmt.Errorf("failed to delete topic: %v, %v", name, err)
			}
		}
	}

	input := &sns.CreateTopicInput{
		Name:       aws.String(name),
		Attributes: make(map[string]*string),
	}

	if resource.Config != nil && len(resource.Config.Attributes) > 0 {
		for k, v := range resource.Config.Attributes {
			input.Attributes[k] = aws.String(v)
		}
	}

	result, err := c.sns.CreateTopic(input)
	if err != nil {
		return nil, err
	}
	var resultResource = &Resource{URL: *result.TopicArn, Name: resource.Name}
	return resultResource, nil
}

func (c *awsClient) SetupResource(resource *ResourceSetup) (*Resource, error) {

	switch resource.Type {
	case ResourceTypeTopic:
		return c.createTopic(resource)
	case ResourceTypeQueue:
		return c.createQueue(resource)
	}
	return nil, fmt.Errorf("unsupported resource type: %v", resource.Type)
}

func (c *awsClient) deleteQueue(resource *Resource) error {
	queueURL, err := c.getQueueURL(resource.Name)
	if err != nil {
		return err
	}
	_, err = c.sqs.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return err
}

func (c *awsClient) deleteTopic(resource *Resource) error {
	queueURL, err := c.getTopicARN(resource.Name)
	if err != nil {
		return err
	}
	_, err = c.sns.DeleteTopic(&sns.DeleteTopicInput{
		TopicArn: aws.String(queueURL),
	})
	return nil
}

func (c *awsClient) DeleteResource(resource *Resource) error {
	switch resource.Type {
	case ResourceTypeQueue:
		return c.deleteQueue(resource)
	case ResourceTypeTopic:
		return c.deleteTopic(resource)
	}
	return fmt.Errorf("unsupported resource type: %v", resource.Type)
}

func (c *awsClient) connect() (err error) {

	if c.session, err = session.NewSession(c.config); err != nil {
		return err
	}
	c.sqs = sqs.New(c.session)
	c.sns = sns.New(c.session)
	return nil
}

func (c *awsClient) Close() error {
	return nil
}


func newAwsSqsClient(credConfig *cred.Config, timeout time.Duration) (Client, error) {
	config, err := eaws.GetAWSCredentialConfig(credConfig)
	if err != nil {
		return nil, err
	}
	config.Region = &credConfig.Region

	var client = &awsClient{
		timeout: timeout,
		config:  config,
	}
	return client, client.connect()
}
