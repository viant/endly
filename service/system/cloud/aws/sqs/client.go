package sqs

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
)

var clientKey = (*sqs.SQS)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sqss := session.Must(session.NewSession())
	client := sqs.New(sqss, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &sqs.SQS{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

// GetClient returns sqs client from context
func GetClient(context *endly.Context) (*sqs.SQS, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	s3Client, ok := client.(*sqs.SQS)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return s3Client, nil
}
