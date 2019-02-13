package sns

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
)

var clientKey = (*sns.SNS)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	snss := session.Must(session.NewSession())
	client := sns.New(snss, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &sns.SNS{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

//GetClient returns sns client from context
func GetClient(context *endly.Context) (*sns.SNS, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	snsClient, ok := client.(*sns.SNS)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return snsClient, nil
}
