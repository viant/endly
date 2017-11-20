package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)


//DeploymentDeployRequest represent a deploy request
type DeploymentDeployRequest struct {
	Target *url.Resource
	MetaURL string //deployment URL for meta deployment instruction
	AppName string //app name
	Version string //requested version
	Force   bool   //flag force deployment, by default if requested version (Transfer.Target.Version is the one from command version check. deployment is skipped.
}

func (r *DeploymentDeployRequest) Validate() error {
	if r.AppName == "" {
		return errors.New("App name was empty")
	}
	return nil
}




//DeploymentDeployResponse represents a deploy response.
type DeploymentDeployResponse struct {
	Version string
}

