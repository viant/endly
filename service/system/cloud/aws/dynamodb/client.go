package dynamodb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
)

var clientKey = (*dynamodb.DynamoDB)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sess := session.Must(session.NewSession())
	client := dynamodb.New(sess, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &dynamodb.DynamoDB{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

// GetClient returns dynamodb client from context
func GetClient(context *endly.Context) (*dynamodb.DynamoDB, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	dynamodbClient, ok := client.(*dynamodb.DynamoDB)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}

	return dynamodbClient, nil
}
