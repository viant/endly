package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//BuildSpec represents build specification.
type BuildSpec struct {
	Name       string `required:"true" description:"build system name, i.e go, mvn, node, yarn, build system meta is defined in meta/build/XXX"`
	Version    string `required:"true" description:"build system version"`
	Goal       string `required:"true" description:"build goal to be matched with build meta goal"`
	BuildGoal  string `required:"true" description:"actual build target, like clean, test"`
	Args       string `required:"true" description:"additional build arguments , that can be expanded with $build.args in build meta"`
	Sdk        string
	SdkVersion string
}

//BuildRequest represents a build request.
type BuildRequest struct {
	MetaURL     string            `description:"build meta URL"`
	BuildSpec   *BuildSpec        `required:"true" description:"build specification" `
	Credentials map[string]string `description:"key value pair of placeholder and credential files, check build meta file for used placeholders i.e for 'go' build: ##git## - git usernamem, **git** - git password"`
	Env         map[string]string `description:"environmental variables"`
	Target      *url.Resource     `required:"true" description:"build location, host and path" `
}

//BuildResponse represents a build response.
type BuildResponse struct {
	CommandInfo *CommandResponse
}

//Validate validates if request is valid
func (r *BuildRequest) Validate() error {
	if r.BuildSpec == nil {
		return errors.New("buildSpec was empty")
	}
	if r.BuildSpec.Name == "" {
		return fmt.Errorf("buildSpec.Name was empty for %v", r.BuildSpec.Name)
	}
	if r.BuildSpec.Goal == "" {
		return fmt.Errorf("buildSpec.Goal was empty for %v", r.BuildSpec.Name)
	}
	return nil
}
