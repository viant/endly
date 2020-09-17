package gcp

import (
	"github.com/viant/endly"
	"google.golang.org/api/cloudresourcemanager/v1"
)


var clientKey = (*AbstractClient)(nil)

//GetProject returns project info
func GetProject(ctx *endly.Context) (*cloudresourcemanager.Project, error)  {
	client := &AbstractClient{}
	err := GetClient(ctx, cloudresourcemanager.New, clientKey, &client, cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	mgr, err  := cloudresourcemanager.New(client.HttpClinet)
	if err != nil {
		return nil, err
	}
	call := mgr.Projects.Get(client.CredConfig.ProjectID)
	return call.Do()
}
