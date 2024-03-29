package kms

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
)

var clientKey = (*kms.KMS)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	kmss := session.Must(session.NewSession())
	client := kms.New(kmss, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &kms.KMS{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

// GetClient returns kms client from context
func GetClient(context *endly.Context) (*kms.KMS, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	kmsClient, ok := client.(*kms.KMS)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return kmsClient, nil
}
