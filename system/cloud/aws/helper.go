package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

//LambdaInvoke represents lambda action permission
var LambdaInvoke = "lambda:InvokeFunction"

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
	if err != nil {
		return nil, err
	}
	ARN, ok := output.Attributes[arnAttribute]
	if !ok {
		return nil, fmt.Errorf("unable to get queue %v ARN", urlOutput.QueueUrl)
	}
	return ARN, nil
}

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

func GetQueueARN(context *endly.Context, name string) (*string, error) {
	client := &sqs.SQS{}
	if err := GetClient(context, dynamodb.New, &client); err != nil {
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
		AttributeNames: []*string{aws.String("QueueArn")},
	})
	if err != nil {
		return nil, err
	}
	return queueAttributes.Attributes["QueueArn"], nil
}
