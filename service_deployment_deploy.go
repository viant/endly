package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DeploymentDeployRequest represent a deploy request
type DeploymentDeployRequest struct {
	Target  *url.Resource `required:"true" description:"target host"`                                                                                   //target host
	MetaURL string        `description:"optional URL for meta deployment file, if left empty the meta URL is construct as meta/deployment/**AppName**"` //deployment URL for meta deployment instruction
	AppName string        `required:"true" description:"application name, as defined in meta deployment file"`                                          //app name
	Version string        `description:"min required version, it can be 1, or 1.2 or specific version 1.2.1"`                                           //requested version
	Force   bool          `description:"force deployment even if app has been already installed"`                                                       //flag force deployment, by default if requested version matches the one from command version check. deployment is skipped.
}

//Validate check if request is valid otherwise returns error.
func (r *DeploymentDeployRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target host was nil")
	}
	if r.AppName == "" {
		return errors.New("app name was empty")
	}
	return nil
}

//DeploymentDeployResponse represents a deploy response.
type DeploymentDeployResponse struct {
	Version string
}
