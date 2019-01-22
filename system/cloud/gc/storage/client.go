package storage

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"google.golang.org/api/storage/v1"
)

var clientKey = (*CtxClient)(nil)

//CtxClient represents context client
type CtxClient struct {
	*gc.AbstractClient
	service *storage.Service
}

func (s *CtxClient) SetService(service interface{}) error {
	var ok bool
	s.service, ok = service.(*storage.Service)
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
	err := gc.GetClient(context, storage.New, clientKey, &client, storage.CloudPlatformScope, storage.PubsubScope)
	return client, err
}
