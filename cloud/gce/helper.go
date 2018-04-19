package gce

import (
	"fmt"
	"github.com/viant/toolbox/cred"
	"golang.org/x/net/context"
	netcontext "golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
	"reflect"
)

//NewComputeService creates a new compute service.
func NewComputeService(credConfig *cred.Config) (*compute.Service, netcontext.Context, error) {
	jwtConfig, err := credConfig.NewJWTConfig(compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, jwtConfig.TokenSource(ctx))
	cilent, err := compute.New(httpClient)
	return cilent, ctx, err
}

//GetComputeService returns specialised compute service for provided service name.
func GetComputeService(client *compute.Service, service string) (interface{}, error) {
	_, found := reflect.TypeOf(*client).FieldByName(service)
	if !found {
		return nil, fmt.Errorf("failed to lookup service %v on google compute service", service)
	}
	return reflect.ValueOf(*client).FieldByName(service).Interface(), nil
}
