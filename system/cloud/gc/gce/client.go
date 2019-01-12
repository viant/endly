package gce

import (
	"fmt"
	"github.com/viant/endly/system/cloud/gc"
	"github.com/viant/toolbox/cred"
	netcontext "golang.org/x/net/context"
	"google.golang.org/api/compute/v1"
	"reflect"
)

//NewComputeService creates a new compute service.
func NewComputeService(credConfig *cred.Config) (*compute.Service, netcontext.Context, error) {
	ctx, httpClient, err := gc.GetGCAuthClient(nil, credConfig, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}
	client, err := compute.New(httpClient)
	return client, ctx, err
}



//GetComputeService returns specialised compute service for provided service name.
func GetComputeService(client *compute.Service, service string) (interface{}, error) {
	_, found := reflect.TypeOf(*client).FieldByName(service)
	if !found {
		return nil, fmt.Errorf("failed to lookup service %v on google compute service", service)
	}
	return reflect.ValueOf(*client).FieldByName(service).Interface(), nil
}
