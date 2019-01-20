package apigateway

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
)



var clientKey = (*apigateway.APIGateway)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sess := session.Must(session.NewSession())
	client :=  apigateway.New(sess, config)
	return context.Put(clientKey, client)
}


func getClient(context *endly.Context) (interface{}, error)  {
	client :=  &apigateway.APIGateway{}
	if ! context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}



//GetClient returns apigateway client from context
func GetClient(context *endly.Context) (*apigateway.APIGateway, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	apigatewayClient, ok := client.(*apigateway.APIGateway)
	if !  ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}
	return apigatewayClient, nil
}