package run

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/run/v1"
)

const baseURLTemplate = "https://%s-run.googleapis.com/"

var clientKey = (*CtxClient)(nil)

//CtxClient represents context client
type CtxClient struct {
	*gcp.AbstractClient
	service *run.APIService
}

func (s *CtxClient) SetService(service interface{}) error {
	var ok bool
	s.service, ok = service.(*run.APIService)
	if !ok {
		return fmt.Errorf("unable to set service: %T", service)
	}
	return nil
}

func (s *CtxClient) Service() interface{} {
	return s.service
}

func InitRequest(context *endly.Context, rawRequest map[string]interface{}) error {
	config, err := gcp.InitCredentials(context, rawRequest)
	if err != nil {
		return err
	}
	client, err := getClient(context)
	if err != nil {
		return err
	}
	gcp.UpdateActionRequest(rawRequest, config, client)
	return nil
}

func getClient(context *endly.Context) (gcp.CtxClient, error) {
	return GetClient(context)
}

func GetClient(context *endly.Context) (*CtxClient, error) {
	client := &CtxClient{
		AbstractClient: &gcp.AbstractClient{},
	}
	//TODO run valid region validation

	err := gcp.GetClient(context, run.New, clientKey, &client, run.CloudPlatformScope)
	if client.service != nil {
		client.service.BasePath = fmt.Sprintf(baseURLTemplate, client.CredConfig.Region)
	}
	return client, err
}
