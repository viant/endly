package cloudwatchevents

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
)

var clientKey = (*cloudwatchevents.CloudWatchEvents)(nil)

func setClient(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := aws.InitCredentials(context, rawRequest, clientKey)
	if err != nil || config == nil {
		return err
	}
	sess := session.Must(session.NewSession())
	client := cloudwatchevents.New(sess, config)
	return context.Put(clientKey, client)
}

func getClient(context *endly.Context) (interface{}, error) {
	client := &cloudwatchevents.CloudWatchEvents{}
	if !context.Contains(clientKey) {
		_ = setClient(context, map[string]interface{}{"client": 1})
	}
	if !context.GetInto(clientKey, &client) {
		return nil, fmt.Errorf("unable to locate client %T, please add Credentials atribute ", client)
	}
	return client, nil
}

//GetClient returns cloudwatchevents client from context
func GetClient(context *endly.Context) (*cloudwatchevents.CloudWatchEvents, error) {
	client, err := getClient(context)
	if err != nil {
		return nil, err
	}
	cloudwatcheventsClient, ok := client.(*cloudwatchevents.CloudWatchEvents)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", client)
	}

	return cloudwatcheventsClient, nil
}
