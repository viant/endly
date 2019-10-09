package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

//LambdaInvoke represents lambda action permission
var LambdaInvoke = "lambda:InvokeFunction"

//SetFunctionInfo sets function info into map
func SetFunctionInfo(function *lambda.FunctionConfiguration, aMap data.Map) {
	functionState := data.NewMap()
	functionState.Put("arn", function.FunctionArn)
	if ARN, err := arn.Parse(*function.FunctionArn); err == nil {
		functionState.Put("region", ARN.Region)
		functionState.Put("accountID", ARN.AccountID)
	}
	functionState.Put("name", function.FunctionName)
	aMap.Put("function", functionState)
}

//GetFunctionConfiguration returns function configuration
func GetFunctionConfiguration(context *endly.Context, functionName string) (*lambda.FunctionConfiguration, error) {
	client := &lambda.Lambda{}
	err := GetClient(context, lambda.New, &client)
	if err != nil {
		return nil, err
	}
	functionOutput, err := client.GetFunction(&lambda.GetFunctionInput{
		FunctionName: &functionName,
	})
	if err != nil {
		return nil, err
	}
	return functionOutput.Configuration, nil
}

//GetKinesisStreamARN returns stream arn
func GetKinesisStreamARN(context *endly.Context, name string) (*string, error) {
	client := &kinesis.Kinesis{}
	err := GetClient(context, kinesis.New, &client)
	if err == nil {
		streamOutput, err := client.DescribeStream(&kinesis.DescribeStreamInput{
			StreamName: &name,
		})
		if err != nil {
			return nil, err
		}
		if streamOutput.StreamDescription == nil {
			return nil, fmt.Errorf("streamDescription was empty ")
		}
		return streamOutput.StreamDescription.StreamARN, nil
	}
	return nil, err
}

//GetKinesisConsumerARN get consumer arn
func GetKinesisConsumerARN(context *endly.Context, name string) (*string, error) {
	client := &kinesis.Kinesis{}
	err := GetClient(context, kinesis.New, &client)
	if err == nil {
		streamOutput, err := client.DescribeStreamConsumer(&kinesis.DescribeStreamConsumerInput{
			ConsumerName: &name,
		})
		if err != nil {
			return nil, err
		}
		if streamOutput.ConsumerDescription == nil {
			return nil, fmt.Errorf("consumerDescription was empty ")
		}
		return streamOutput.ConsumerDescription.ConsumerARN, nil
	}
	return nil, err
}

//GetSqsURL returns sqs URL
func GetSqsURL(context *endly.Context, name string) (*string, error) {
	client := &sqs.SQS{}
	if err := GetClient(context, sqs.New, &client); err != nil {
		return nil, err
	}
	urlOutput, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &name,
	})
	if err != nil {
		return nil, err
	}
	var arnAttribute = "QueueArn"
	output, err := client.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       urlOutput.QueueUrl,
		AttributeNames: []*string{&arnAttribute},
	})
	if err != nil {
		return nil, err
	}
	ARN, ok := output.Attributes[arnAttribute]
	if !ok {
		return nil, fmt.Errorf("unable to get queue %v ARN", urlOutput.QueueUrl)
	}
	return ARN, nil
}

//GetDynamoDBTableARN returns dynamo db ARN
func GetDynamoDBTableARN(context *endly.Context, name string) (*string, error) {
	client := &dynamodb.DynamoDB{}
	if err := GetClient(context, dynamodb.New, &client); err != nil {
		return nil, err
	}
	output, err := client.DescribeGlobalTable(&dynamodb.DescribeGlobalTableInput{
		GlobalTableName: &name,
	})

	if err != nil {
		return nil, err
	}
	if output.GlobalTableDescription == nil {
		return nil, fmt.Errorf("globalTableDescription was empty")
	}
	return output.GlobalTableDescription.GlobalTableArn, nil
}

//GetQueueARN returns qeueue arn
func GetQueueARN(context *endly.Context, name string) (*string, error) {
	client := &sqs.SQS{}
	if err := GetClient(context, sqs.New, &client); err != nil {
		return nil, err
	}
	output, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &name,
	})
	if err != nil {
		return nil, err
	}
	queueAttributes, err := client.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       output.QueueUrl,
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNameQueueArn)},
	})
	if err != nil {
		return nil, err
	}
	return queueAttributes.Attributes[sqs.QueueAttributeNameQueueArn], nil
}

//GetTopicARN returns topic arn
func GetTopicARN(context *endly.Context, name string) (*string, error) {
	client := &sns.SNS{}
	if err := GetClient(context, sns.New, &client); err != nil {
		return nil, err
	}

	var nextToken *string
	for {
		output, err := client.ListTopics(&sns.ListTopicsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, topic := range output.Topics {
			ARN, _ := arn.Parse(*topic.TopicArn)
			if ARN.Resource == name {
				return topic.TopicArn, nil
			}
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil, fmt.Errorf("failed to lookup topic: %v", name)
}

//NextID return new ID
func NextID() (string, error) {
	UUID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	uuidValue := UUID.String()
	return uuidValue, nil
}
