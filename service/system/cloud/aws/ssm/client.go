package ssm

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
)

var clientKey = (*ssm.SSM)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	ssms := session.Must(session.NewSession())
	client := ssm.New(ssms, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &ssm.SSM{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

// GetClient returns ssm client from context
func GetClient(context *endly.Context) (*ssm.SSM, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	ssmClient, ok := client.(*ssm.SSM)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return ssmClient, nil
}
