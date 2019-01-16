package cloudwatch

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
)


var clientKey = (*cloudwatch.CloudWatch)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitAws(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sess := session.Must(session.NewSession())
	client :=  cloudwatch.New(sess, config)
	return context.Put(clientKey, client)
}


func getClient(context *endly.Context) (interface{}, error)  {
	client :=  &cloudwatch.CloudWatch{}
	if ! context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}


//GetClient returns cloudwatch client from context
func GetClient(context *endly.Context) (*cloudwatch.CloudWatch, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	cloudwatchClient, ok := client.(*cloudwatch.CloudWatch)
	if !  ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}

	return cloudwatchClient, nil
}