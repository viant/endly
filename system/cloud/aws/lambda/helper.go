package lambda

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/viant/toolbox/data"
)

func hasDataChanged(data []byte, dataSha256 string) bool {
	algorithm := sha256.New()
	algorithm.Write(data)
	codeSha1 := algorithm.Sum(nil)
	dataSha1BAse64Encoded := base64.URLEncoding.EncodeToString(codeSha1)
	return dataSha256 != dataSha1BAse64Encoded
}

func GetFunctionConfiguration(context *endly.Context, functionName string) (*lambda.FunctionConfiguration,  error) {
	client, err := GetClient(context)
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


func getKinesisStreamARN(context *endly.Context, name string) (*string, error) {
	client := &kinesis.Kinesis{}
	err := aws.GetClient(context, kinesis.New, &client);
	if err == nil {
		streamOutput, err := client.DescribeStream(&kinesis.DescribeStreamInput{
			StreamName:&name,
		});
		if err != nil  {
			return nil, err
		}
		if streamOutput.StreamDescription == nil {
			return nil, fmt.Errorf("streamDescription was empty ")
		}
		return streamOutput.StreamDescription.StreamARN, nil
	}
	return nil, err
}

func getKinesisConsumerARN(context *endly.Context, name string) (*string, error) {
	client := &kinesis.Kinesis{}
	err := aws.GetClient(context, kinesis.New, &client);
	if err == nil {
		streamOutput, err := client.DescribeStreamConsumer(&kinesis.DescribeStreamConsumerInput{
			ConsumerName:&name,
		});
		if err != nil  {
			return nil, err
		}
		if streamOutput.ConsumerDescription == nil {
			return nil, fmt.Errorf("consumerDescription was empty ")
		}
		return streamOutput.ConsumerDescription.ConsumerARN, nil
	}
	return nil, err
}

func getSqsURL(context *endly.Context, name string) (*string, error) {
	client := &sqs.SQS{}
	if err := aws.GetClient(context, sqs.New, &client); err != nil {
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
		QueueUrl: urlOutput.QueueUrl,
		AttributeNames:[]*string{&arnAttribute},
	})
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	ARN, ok :=  output.Attributes[arnAttribute]
	if ! ok {
		return nil, fmt.Errorf("unable to get queue %v ARN", urlOutput.QueueUrl)
	}
	return ARN, nil
}


func getDynamoDBTableARN(context *endly.Context, name string) (*string, error) {
	client := &dynamodb.DynamoDB{}
	if err := aws.GetClient(context, dynamodb.New, &client); err != nil {
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
	return  output.GlobalTableDescription.GlobalTableArn, nil
}

