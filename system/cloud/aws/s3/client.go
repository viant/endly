package s3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
)

var clientKey = (*s3.S3)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sess := session.Must(session.NewSession())
	client := s3.New(sess, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &s3.S3{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

// GetClient returns s3 client from context
func GetClient(context *endly.Context) (*s3.S3, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	s3Client, ok := client.(*s3.S3)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return s3Client, nil
}
