package pubsub

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"google.golang.org/api/pubsub/v1"
)

var clientKey = (*CtxClient)(nil)

//CtxClient represents context client
type CtxClient struct {
	*gc.AbstractClient
	service *pubsub.Service
}

func (s *CtxClient) SetService(service interface{}) error {
	var ok bool
	s.service, ok = service.(*pubsub.Service)
	if ! ok {
		return fmt.Errorf("unable to set service: %T", service)
	}
	return nil
}

func (s *CtxClient) Service() interface{} {
	return s.service
}

func InitRequest(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := gc.InitCredentials(context, rawRequest)
	if err != nil {
		return err
	}
	client, err := getClient(context)
	if err != nil {
		return err
	}
	gc.UpdateActionRequest(rawRequest, config, client)
	return nil
}

func getClient(context *endly.Context) (gc.CtxClient, error) {
	return GetClient(context)
}

func GetClient(context *endly.Context) (*CtxClient, error) {
	client := &CtxClient{
		AbstractClient: &gc.AbstractClient{},
	}
	err := gc.GetClient(context, pubsub.New, clientKey, &client, pubsub.CloudPlatformScope, pubsub.PubsubScope)
	return client, err
}
